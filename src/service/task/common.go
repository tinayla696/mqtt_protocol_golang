// service/process/common.go
// DEV: Workerプロセスの共通部の定義
package task

import "context"

// *--------------------------------------------------------------------------------------
// Task
// DEV: interface{} == Workerが実行するTaskを抽象化したもの
type Task interface {
	Execute(ctx context.Context) error
	String() string // Task識別子
	Type() TaskType // Task種別
}

// *--------------------------------------------------------------------------------------
// TaskType
// DEV: タスクの種別を定義
type TaskType string

const (
	MqttTaskType  TaskType = "mqtt"  // MQTTタスク
	OtherTaskType TaskType = "other" // その他のタスク (例: HTTP, DB, etc.
)
