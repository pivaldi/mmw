package defauth

import "time"

// The Topic Name (Routing Key)
const TopicUserDeleted = "auth.user.deleted.v1"

// Define the Event Payload DTO
// TODO: migrate to Protobuf, this struct should be swapped for the generated proto type
type UserDeletedEvent struct {
	UserID    string    `json:"userId"`
	DeletedAt time.Time `json:"deletedAt"`
}
