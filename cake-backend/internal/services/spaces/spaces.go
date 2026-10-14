// Package spaces implements the business logic for user-owned conversation spaces.
package spaces

import (
	"context"
	"efournierrobert/cake-backend/internal/encryptor"
	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	"efournierrobert/cake-backend/internal/oai"
	convosRepo "efournierrobert/cake-backend/internal/repository/conversations"
	messagesRepo "efournierrobert/cake-backend/internal/repository/messages"
	modelsRepo "efournierrobert/cake-backend/internal/repository/models"
	"efournierrobert/cake-backend/internal/repository/providers"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
	spaceRepo "efournierrobert/cake-backend/internal/repository/spaces"
	userRepo "efournierrobert/cake-backend/internal/repository/users"
	"errors"
	"log"
	"uuid"

	"github.com/jmoiron/sqlx"
)

// Service applies ownership rules and maps persisted spaces to API DTOs.
type Service struct {
	repo          spaceRepo.SpaceRepository
	userRepo      userRepo.UserRepository
	messagesRepo  messagesRepo.MessagesRepo
	convosRepo    convosRepo.ConvosRepo
	modelsRepo    modelsRepo.ModelRepository
	providersRepo *providers.Repository
	oai           *oai.OpenAi
	encryptor     *encryptor.Encryptor
}

// New creates a spaces service backed by db.
func New(db *sqlx.DB, encryptor *encryptor.Encryptor) *Service {
	return &Service{
		repo:          spaceRepo.New(db),
		userRepo:      userRepo.New(db),
		messagesRepo:  messagesRepo.New(db),
		convosRepo:    convosRepo.New(db),
		modelsRepo:    modelsRepo.New(db),
		providersRepo: providers.New(db),
		oai:           oai.New(),
		encryptor:     encryptor,
	}
}

// GetAllSpaces returns every space owned by ownerUuidStr. An empty result is
// returned as a non-nil empty slice. Invalid owner UUIDs return
// handler_errors.ErrInvalidRequest.
func (s *Service) GetAllSpaces(ownerUuidStr string) ([]models.SpaceDto, error) {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return nil, handler_errors.ErrInvalidRequest
	}

	allUserSpaces, err := s.repo.GetAllUserSpaces(ownerUuid)
	if err != nil {
		return nil, getAppErrorType(err)
	}

	dto := []models.SpaceDto{}
	for _, s := range allUserSpaces {
		dto = append(dto, spaceToDto(s))
	}

	return dto, nil
}

// GetSpace returns the space identified by spaceUuidStr, provided it belongs
// to ownerUuidStr. Invalid UUIDs return handler_errors.ErrInvalidRequest and
// missing or non-owned spaces return handler_errors.ErrSpaceNotFound.
func (s *Service) GetSpace(spaceUuidStr, ownerUuidStr string) (models.SpaceDto, error) {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return models.SpaceDto{}, handler_errors.ErrInvalidRequest
	}

	spaceUuid, err := uuid.Parse(spaceUuidStr)
	if err != nil {
		return models.SpaceDto{}, handler_errors.ErrInvalidRequest
	}

	space, err := s.repo.GetUserSpace(ownerUuid, spaceUuid)
	if err != nil {
		return models.SpaceDto{}, getAppErrorType(err)
	}

	return spaceToDto(space), nil
}

// CreateSpace creates a space owned by ownerUuidStr and returns its DTO. The
// owner must identify an existing user. The system prompt is stored as a
// string, including an empty string when omitted.
func (s *Service) CreateSpace(ownerUuidStr string, spaceToCreate models.SpaceCreate) (models.SpaceDto, error) {
	user, err := s.getUserFromUuid(ownerUuidStr)
	if err != nil {
		return models.SpaceDto{}, err
	}

	space := spaceRepo.Space{
		Uuid:         uuid.NewV4().String(),
		Name:         spaceToCreate.Name,
		SystemPrompt: spaceToCreate.SystemPrompt,
		OwnerId:      user.Id,
	}

	space, err = s.repo.CreateSpace(space)
	if err != nil {
		return models.SpaceDto{}, getAppErrorType(err)
	}

	return spaceToDto(space), nil
}

// UpdateSpace applies the non-nil fields in spaceUpdate to the owner's space
// and returns the updated DTO. A nil field is left unchanged; an explicitly
// supplied empty string is stored as an empty string.
func (s *Service) UpdateSpace(ownerUuidStr, spaceUuidStr string, spaceUpdate models.SpaceUpdate) (models.SpaceDto, error) {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return models.SpaceDto{}, handler_errors.ErrInvalidRequest
	}

	spaceUuid, err := uuid.Parse(spaceUuidStr)
	if err != nil {
		return models.SpaceDto{}, handler_errors.ErrInvalidRequest
	}

	space, err := s.repo.GetUserSpace(ownerUuid, spaceUuid)
	if err != nil {
		return models.SpaceDto{}, getAppErrorType(err)
	}

	if spaceUpdate.Name != nil {
		space.Name = *spaceUpdate.Name
	}
	if spaceUpdate.SystemPrompt != nil {
		space.SystemPrompt = *spaceUpdate.SystemPrompt
	}

	space, err = s.repo.UpdateSpace(space)
	if err != nil {
		return models.SpaceDto{}, getAppErrorType(err)
	}

	return spaceToDto(space), nil
}

// DeleteSpace removes the space identified by spaceUuidStr when it belongs to
// ownerUuidStr. Invalid UUIDs return handler_errors.ErrInvalidRequest and a
// missing or non-owned space returns handler_errors.ErrSpaceNotFound.
func (s *Service) DeleteSpace(ownerUuidStr, spaceUuidStr string) error {
	ownerUuid, err := uuid.Parse(ownerUuidStr)
	if err != nil {
		return handler_errors.ErrInvalidRequest
	}

	spaceUuid, err := uuid.Parse(spaceUuidStr)
	if err != nil {
		return handler_errors.ErrInvalidRequest
	}

	err = s.repo.DeleteSpace(ownerUuid, spaceUuid)
	if err != nil {
		return getAppErrorType(err)
	}

	return nil
}

func (s *Service) CreateChat(
	ctx context.Context,
	ownerUuidStr,
	spaceUuidStr string,
	message models.ChatCreate) (<-chan models.ChatDto, error) {
	owner, err := s.getUserFromUuid(ownerUuidStr)
	if err != nil {
		return nil, err
	}

	space, err := s.getSpaceFromUuid(uuid.MustParse(owner.Uuid), spaceUuidStr)
	if err != nil {
		return nil, err
	}

	model, err := s.getModelFromUuid(message.ModelUuid)
	if err != nil {
		return nil, err
	}

	provider, err := s.providersRepo.GetProviderById(model.ProviderId)
	if err != nil {
		return nil, getAppErrorType(err)
	}

	apiKey := []byte{}
	if len(provider.ApiKey) > 0 {
		apiKey, err = s.encryptor.Decrypt(provider.ApiKey)
		if err != nil {
			return nil, getAppErrorType(err)
		}
	}

	newMessage := messagesRepo.Message{
		Uuid:           uuid.NewV4().String(),
		Content:        message.Message,
		ConversationId: 0,
		RoleId:         0,
	}

	convo, err := s.ensureConversation(message.ConversationUuid, owner.Id, space.ID, uuid.MustParse(owner.Uuid))
	if err != nil {
		return nil, err
	}

	newMessage.ConversationId = convo.ID

	role, err := s.messagesRepo.GetUserRole()
	if err != nil {
		return nil, getAppErrorType(err)
	}

	newMessage.RoleId = role.ID

	err = s.messagesRepo.CreateMessage(newMessage)
	if err != nil {
		return nil, getAppErrorType(err)
	}

	wholeConvo, err := s.messagesRepo.GetWholeConversation(convo.ID)
	if err != nil {
		return nil, getAppErrorType(err)
	}

	var oaiInput []oai.ResponseInput
	for _, m := range wholeConvo {
		oaiInput = append(oaiInput, oai.ResponseInput{
			Role:    m.RoleName,
			Content: m.Content,
		})
	}

	oaiChannel, err := s.oai.CreateResponse(
		ctx,
		string(apiKey),
		provider.BaseUrl,
		model.ProviderModelId,
		space.SystemPrompt,
		oaiInput)
	if err != nil {
		return nil, getAppErrorType(err)
	}

	outputChannel := make(chan models.ChatDto)

	go func() {
		defer close(outputChannel)
		outputChannel <- models.ChatDto{
			Event: "response.start",
			Data:  models.ChatDtoData{},
		}

		for data := range oaiChannel {
			if data.Error != nil {
				log.Printf("error during openAI request: %v", err)
				outputChannel <- models.ChatDto{
					Error: "error during request",
				}
			}

			switch data.Data.Type {
			case "response.output_text.delta":
				select {
				case outputChannel <- models.ChatDto{
					Event: data.Data.Type,
					Data: models.ChatDtoData{
						MessageUuid: nil,
						Content:     &data.Data.Delta,
					},
				}:
				case <-ctx.Done():
				}
			case "response.completed":
				select {
				case outputChannel <- models.ChatDto{
					Event: data.Data.Type,
					Data: models.ChatDtoData{
						MessageUuid: nil,
						Content:     nil,
					},
				}:
				case <-ctx.Done():
				}
			case "response.failed", "response.incomplete":
				select {
				case outputChannel <- models.ChatDto{
					Event: data.Data.Type,
				}:
				case <-ctx.Done():
				}
				return
			}
		}
	}()

	return outputChannel, nil
}

func (s *Service) ensureConversation(convoUuid *string, ownerId, spaceId int, ownerUuid uuid.UUID) (convosRepo.Conversation, error) {
	var convo convosRepo.Conversation
	var err error

	if convoUuid == nil {
		convo, err = s.convosRepo.CreateConversation(ownerUuid, convosRepo.Conversation{
			Uuid:    uuid.NewV4().String(),
			OwnerId: ownerId,
			Title:   "test convo",
			SpaceId: spaceId,
		})
		if err != nil {
			return convosRepo.Conversation{}, getAppErrorType(err)
		}

	} else {
		convoUuid, err := uuid.Parse(*convoUuid)
		if err != nil {
			return convosRepo.Conversation{}, handler_errors.ErrInvalidRequest
		}

		convo, err = s.convosRepo.GetConversation(ownerUuid, convoUuid)
		if err != nil {
			return convosRepo.Conversation{}, getAppErrorType(err)
		}
	}

	return convo, nil
}

func (s *Service) getModelFromUuid(uuidStr string) (modelsRepo.Model, error) {
	userUuid, err := uuid.Parse(uuidStr)
	if err != nil {
		return modelsRepo.Model{}, handler_errors.ErrInvalidRequest
	}

	model, err := s.modelsRepo.GetModel(userUuid)
	if err != nil {
		return modelsRepo.Model{}, getAppErrorType(err)
	}

	return model, nil
}

func (s *Service) getUserFromUuid(uuidStr string) (userRepo.User, error) {
	userUuid, err := uuid.Parse(uuidStr)
	if err != nil {
		return userRepo.User{}, handler_errors.ErrInvalidRequest
	}

	user, err := s.userRepo.GetUser(userUuid)
	if err != nil {
		return userRepo.User{}, getAppErrorType(err)
	}

	return user, nil
}

func (s *Service) getSpaceFromUuid(ownerUuid uuid.UUID, uuidStr string) (spaceRepo.Space, error) {
	spaceUuid, err := uuid.Parse(uuidStr)
	if err != nil {
		return spaceRepo.Space{}, handler_errors.ErrInvalidRequest
	}

	space, err := s.repo.GetUserSpace(ownerUuid, spaceUuid)
	if err != nil {
		return spaceRepo.Space{}, getAppErrorType(err)
	}

	return space, nil
}

func spaceToDto(s spaceRepo.Space) models.SpaceDto {
	return models.SpaceDto{
		Uuid:         s.Uuid,
		Name:         s.Name,
		SystemPrompt: s.SystemPrompt,
		CreatedAt:    s.CreatedAt,
		UpdatedAt:    s.UpdatedAt,
	}
}

// getAppErrorType maps repository errors to handler errors.
func getAppErrorType(err error) error {
	log.Println("error happened: " + err.Error())

	if _, ok := errors.AsType[*repo_errors.SpaceNotFound](err); ok {
		return handler_errors.ErrSpaceNotFound
	}

	if _, ok := errors.AsType[*repo_errors.UserNotFound](err); ok {
		return handler_errors.ErrUserDoesNotExist
	}
	if _, ok := errors.AsType[*repo_errors.ModelNotFound](err); ok {
		return handler_errors.ErrModelNotFound
	}
	if _, ok := errors.AsType[*repo_errors.ProviderNotFound](err); ok {
		return handler_errors.ErrProviderNotFound
	}
	return handler_errors.ErrUnexpectedError
}
