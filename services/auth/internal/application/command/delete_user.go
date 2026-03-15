package command

import (
	"context"
	"encoding/json"
	"time"

	"github.com/ovya/ogl/platform/pfevents"
	"github.com/pivaldi/mmw/contracts/definitions/auth"
	"github.com/rotisserie/eris"
)

type DeleteUserCommand struct {
	bus pfevents.SystemEventBus
	// ... repo, etc.
}

func NewDeleteUserCommand(bus pfevents.SystemEventBus) *DeleteUserCommand {
	return &DeleteUserCommand{bus: bus}
}

func (c *DeleteUserCommand) Execute(ctx context.Context, userID string) error {
	// Delete user. TODO: from the Auth database
	// c.repo.DeleteUser(ctx, userID)

	// Map the domain event to the Public Contract DTO
	eventDTO := auth.UserDeletedEvent{
		UserID:    userID,
		DeletedAt: time.Now().UTC().Format(time.RFC3339),
	}

	// Serialize the payload. TODO: use Protobuf
	payload, err := json.Marshal(eventDTO)
	if err != nil {
		return eris.Wrap(err, "serializing paylod failed")
	}

	return eris.Wrap(c.bus.Publish(ctx, auth.TopicUserDeleted, payload), "publishing eventbus failed")
}
