// Package conversations implements conversation business logic and maps
// persisted conversations and messages to API DTOs.
package conversations

import (
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	convosRepo "efournierrobert/cake-backend/internal/repository/conversations"
	"efournierrobert/cake-backend/internal/repository/messages"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	"errors"
	"log"
	"uuid"

	"github.com/jmoiron/sqlx"
)

// Service applies conversation ownership rules and maps repository records to
// API DTOs. Conversation lookups and mutations are always scoped to an owner
// and space.
type Service struct {
	repo         convosRepo.ConvosRepo
	messagesRepo messages.MessagesRepo
}

// New creates a conversation service backed by db.
func New(db *sqlx.DB) *Service {
	return &Service{
		repo:         convosRepo.New(db),
		messagesRepo: messages.New(db),
	}
}

// GetConversations returns conversations belonging to ownerUuidStr in
// spaceUuidStr. Invalid UUIDs return ErrInvalidRequest; repository failures
// return ErrUnexpectedError. An empty result is a non-nil empty slice.
func (s *Service) GetConversations(ownerUuidStr, spaceUuidStr string) ([]models.ConversationDto, error) {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return nil, handler_errors.ErrInvalidRequest
	}

	spaceUuid, err := uuid.Parse(spaceUuidStr)
	if err != nil {
		return nil, handler_errors.ErrInvalidRequest
	}

	convos, err := s.repo.GetSpaceConversations(ownerUuid, spaceUuid)
	if err != nil {
		log.Printf("GetConversations error: %v\n", err)
		return nil, &handler_errors.ErrUnexpectedError
	}

	dto := make([]models.ConversationDto, 0)
	for _, c := range convos {
		dto = append(dto, convoToDto(c))
	}

	return dto, nil
}

// GetConversation returns the conversation identified by convoUuidStr when it
// belongs to the supplied owner and space. Invalid UUIDs return
// ErrInvalidRequest, missing or out-of-scope conversations return
// ErrConvoNotFound, and other repository failures return ErrUnexpectedError.
func (s *Service) GetConversation(ownerUuidStr, spaceUuidStr, convoUuidStr string) (models.ConversationDto, error) {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return models.ConversationDto{}, handler_errors.ErrInvalidRequest
	}

	spaceUuid, err := uuid.Parse(spaceUuidStr)
	if err != nil {
		return models.ConversationDto{}, handler_errors.ErrInvalidRequest
	}

	convoUuid, err := uuid.Parse(convoUuidStr)
	if err != nil {
		return models.ConversationDto{}, handler_errors.ErrInvalidRequest
	}

	convo, err := s.repo.GetConversation(ownerUuid, spaceUuid, convoUuid)
	if err != nil {
		log.Printf("GetConversation error: %v\n", err)
		return models.ConversationDto{}, getAppErrorType(err)
	}

	return convoToDto(convo), nil
}

// UpdateConversation changes the title of a conversation belonging to the
// supplied owner and space, then returns the updated DTO. UUID and repository
// errors are mapped to their corresponding handler errors.
func (s *Service) UpdateConversation(ownerUuidStr, spaceUuidStr, convoUuidStr, newTitle string) (models.ConversationDto, error) {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return models.ConversationDto{}, handler_errors.ErrInvalidRequest
	}

	spaceUuid, err := uuid.Parse(spaceUuidStr)
	if err != nil {
		return models.ConversationDto{}, handler_errors.ErrInvalidRequest
	}

	convoUuid, err := uuid.Parse(convoUuidStr)
	if err != nil {
		return models.ConversationDto{}, handler_errors.ErrInvalidRequest
	}

	convo, err := s.repo.GetConversation(ownerUuid, spaceUuid, convoUuid)
	if err != nil {
		log.Printf("UpdateConversation error: %v\n", err)
		return models.ConversationDto{}, getAppErrorType(err)
	}

	convo.Title = newTitle

	convo, err = s.repo.UpdateConversation(ownerUuid, spaceUuid, convo)
	if err != nil {
		log.Printf("UpdateConversation error: %v\n", err)
		return models.ConversationDto{}, getAppErrorType(err)
	}

	return convoToDto(convo), nil
}

// DeleteConversation removes the conversation identified by convoUuidStr when
// it belongs to the supplied owner and space. Invalid UUIDs return
// ErrInvalidRequest, missing or out-of-scope conversations return
// ErrConvoNotFound, and other repository failures return ErrUnexpectedError.
func (s *Service) DeleteConversation(ownerUuidStr, spaceUuidStr, convoUuidStr string) error {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return handler_errors.ErrInvalidRequest
	}

	spaceUuid, err := uuid.Parse(spaceUuidStr)
	if err != nil {
		return handler_errors.ErrInvalidRequest
	}

	convoUuid, err := uuid.Parse(convoUuidStr)
	if err != nil {
		return handler_errors.ErrInvalidRequest
	}

	if err := s.repo.DeleteConversation(ownerUuid, spaceUuid, convoUuid); err != nil {
		log.Printf("DeleteConversation error: %v\n", err)
		return getAppErrorType(err)
	}

	return nil
}

// GetFullConversation returns messages for a conversation that belongs to the
// supplied owner and space, in the order provided by the message repository.
// Invalid UUIDs, missing conversations, and repository failures return their
// corresponding handler errors. An empty conversation returns a non-nil empty
// slice.
func (s *Service) GetFullConversation(ownerUuidStr, spaceUuidStr, convoUuidStr string) ([]models.ConversationMessagesDto, error) {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return nil, handler_errors.ErrInvalidRequest
	}

	spaceUuid, err := uuid.Parse(spaceUuidStr)
	if err != nil {
		return nil, handler_errors.ErrInvalidRequest
	}

	convoUuid, err := uuid.Parse(convoUuidStr)
	if err != nil {
		return nil, handler_errors.ErrInvalidRequest
	}

	conversation, err := s.repo.GetConversation(ownerUuid, spaceUuid, convoUuid)
	if err != nil {
		return nil, getAppErrorType(err)
	}

	fullConversation, err := s.messagesRepo.GetWholeConversation(conversation.ID)
	if err != nil {
		return nil, getAppErrorType(err)
	}

	return fullConversationToDto(fullConversation), nil
}

func fullConversationToDto(fullConvo []messages.MessageWithRoleName) []models.ConversationMessagesDto {
	dto := make([]models.ConversationMessagesDto, 0, len(fullConvo))
	for _, m := range fullConvo {
		dto = append(dto, models.ConversationMessagesDto{
			Uuid:      m.Uuid,
			Content:   m.Content,
			CreatedAt: m.CreatedAt,
			RoleName:  m.RoleName,
		})
	}

	return dto
}

func convoToDto(c convosRepo.Conversation) models.ConversationDto {
	return models.ConversationDto{
		Uuid:      c.Uuid,
		Title:     c.Title,
		CreatedAt: c.CreatedAt,
	}
}

// getAppErrorType maps repository errors to handler errors.
func getAppErrorType(err error) error {
	log.Println("error happened: " + err.Error())

	if _, ok := errors.AsType[*repo_errors.ConversationNotFound](err); ok {
		return handler_errors.ErrConvoNotFound
	}
	return handler_errors.ErrUnexpectedError
}
