package task

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/mitchellh/mapstructure"
)

const (
	TagNameTask       = "task"
	TagNameConfig     = "config"
	TagNameCheckpoint = "checkpoint"
)

// TaskInfo 是任务持久化信息，包含任务配置和断点数据。
type TaskInfo struct {
	Config     map[string]any `mapstructure:"config"`
	Checkpoint map[string]any `mapstructure:"checkpoint"`
}

// MarshalTask 将任务结构体拆分为配置和断点数据。
func MarshalTask(v any) (TaskInfo, error) {
	configValue, err := taskSectionValue(v, TagNameConfig)
	if err != nil {
		return TaskInfo{}, fmt.Errorf("find task config: %w", err)
	}
	config, err := MarshalConfig(configValue)
	if err != nil {
		return TaskInfo{}, fmt.Errorf("marshal task config: %w", err)
	}

	checkpointValue, err := taskSectionValue(v, TagNameCheckpoint)
	if err != nil {
		return TaskInfo{}, fmt.Errorf("find task checkpoint: %w", err)
	}
	checkpoint, err := MarshalCheckpoint(checkpointValue)
	if err != nil {
		return TaskInfo{}, fmt.Errorf("marshal task checkpoint: %w", err)
	}
	return TaskInfo{
		Config:     config,
		Checkpoint: checkpoint,
	}, nil
}

// UnmarshalTask 将任务持久化信息写入任务结构体指针。
func UnmarshalTask(info TaskInfo, v any) error {
	configValue, err := taskSectionPointer(v, TagNameConfig)
	if err != nil {
		return fmt.Errorf("find task config: %w", err)
	}
	if err := UnmarshalConfig(info.Config, configValue); err != nil {
		return fmt.Errorf("unmarshal config: %w", err)
	}

	checkpointValue, err := taskSectionPointer(v, TagNameCheckpoint)
	if err != nil {
		return fmt.Errorf("find task checkpoint: %w", err)
	}
	if err := UnmarshalCheckpoint(info.Checkpoint, checkpointValue); err != nil {
		return fmt.Errorf("unmarshal checkpoint: %w", err)
	}
	return nil
}

// MarshalConfig 从任务结构体中导出 config tag 标记的配置字段。
func MarshalConfig(v any) (map[string]any, error) {
	return marshalWithTag(v, TagNameConfig)
}

// UnmarshalConfig 将配置数据写入 config tag 标记的字段。
func UnmarshalConfig(data map[string]any, v any) error {
	return unmarshalWithTag(data, v, TagNameConfig)
}

// MarshalCheckpoint 从任务结构体中导出 checkpoint tag 标记的断点字段。
func MarshalCheckpoint(v any) (map[string]any, error) {
	return marshalWithTag(v, TagNameCheckpoint)
}

// UnmarshalCheckpoint 将断点数据写入 checkpoint tag 标记的字段。
func UnmarshalCheckpoint(data map[string]any, v any) error {
	return unmarshalWithTag(data, v, TagNameCheckpoint)
}

func marshalWithTag(v any, tagName string) (map[string]any, error) {
	result := make(map[string]any)
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
		Metadata:         nil,
		Result:           &result,
		Squash:           true,
		TagName:          tagName,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return nil, err
	}
	if err := decoder.Decode(v); err != nil {
		return nil, err
	}
	return result, nil
}

func unmarshalWithTag(data map[string]any, v any, tagName string) error {
	decoder, err := mapstructure.NewDecoder(&mapstructure.DecoderConfig{
		DecodeHook:       mapstructure.StringToTimeDurationHookFunc(),
		Metadata:         nil,
		Result:           v,
		Squash:           true,
		TagName:          tagName,
		WeaklyTypedInput: true,
	})
	if err != nil {
		return err
	}
	return decoder.Decode(data)
}

func taskSectionValue(v any, name string) (any, error) {
	value := reflect.ValueOf(v)
	if !value.IsValid() {
		return nil, fmt.Errorf("nil task")
	}
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, fmt.Errorf("nil task pointer")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("task must be a struct or struct pointer")
	}

	valueType := value.Type()
	for i := range value.NumField() {
		field := valueType.Field(i)
		tag := tagName(field.Tag.Get(TagNameTask))
		if tag != name {
			continue
		}
		fieldValue := value.Field(i)
		return fieldValue.Interface(), nil
	}
	return nil, fmt.Errorf("task section %q not found", name)
}

func taskSectionPointer(v any, name string) (any, error) {
	value := reflect.ValueOf(v)
	if !value.IsValid() {
		return nil, fmt.Errorf("nil task")
	}
	if value.Kind() != reflect.Pointer || value.IsNil() {
		return nil, fmt.Errorf("task must be a non-nil struct pointer")
	}
	for value.Kind() == reflect.Pointer {
		if value.IsNil() {
			return nil, fmt.Errorf("nil task pointer")
		}
		value = value.Elem()
	}
	if value.Kind() != reflect.Struct {
		return nil, fmt.Errorf("task must point to a struct")
	}

	valueType := value.Type()
	for i := range value.NumField() {
		field := valueType.Field(i)
		tag := tagName(field.Tag.Get(TagNameTask))
		if tag != name {
			continue
		}
		fieldValue := value.Field(i)
		if !fieldValue.CanAddr() {
			return nil, fmt.Errorf("task section %q is not addressable", name)
		}
		return fieldValue.Addr().Interface(), nil
	}
	return nil, fmt.Errorf("task section %q not found", name)
}

func tagName(tag string) string {
	if index := strings.IndexByte(tag, ','); index >= 0 {
		return tag[:index]
	}
	return tag
}
