package deftodo

import "time"

// The Topic Name (Routing Key)
const TopicUserTasksDeleted = "todo.userTasks.deleted.v1"

// Define the Event Payload DTO
// TODO: migrate to Protobuf, this struct should be swapped for the generated proto type
type UserTasksDeletedEvent struct {
	UserID    string    `json:"userId"`
	TasksIDS  []int     `json:"tasksIds"`
	DeletedAt time.Time `json:"deletedAt"`
}
