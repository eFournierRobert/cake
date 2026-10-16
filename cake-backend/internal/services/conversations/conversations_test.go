package conversations

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
	"uuid"

	"efournierrobert/cake-backend/internal/handlers/handler_errors"
	"efournierrobert/cake-backend/internal/handlers/models"
	convosRepo "efournierrobert/cake-backend/internal/repository/conversations"
	"efournierrobert/cake-backend/internal/repository/messages"
	"efournierrobert/cake-backend/internal/repository/repo_errors"
)

const (
	testOwnerUUID = "11111111-1111-4111-8111-111111111111"
	testSpaceUUID = "22222222-2222-4222-8222-222222222222"
	testConvoUUID = "33333333-3333-4333-8333-333333333333"
)

type conversationsRepoStub struct {
	getSpaceConversations func(uuid.UUID, uuid.UUID) ([]convosRepo.Conversation, error)
	getConversation       func(uuid.UUID, uuid.UUID, uuid.UUID) (convosRepo.Conversation, error)
	createConversation    func(uuid.UUID, uuid.UUID, convosRepo.Conversation) (convosRepo.Conversation, error)
	updateConversation    func(uuid.UUID, uuid.UUID, convosRepo.Conversation) (convosRepo.Conversation, error)
	deleteConversation    func(uuid.UUID, uuid.UUID, uuid.UUID) error
}

func (r *conversationsRepoStub) GetSpaceConversations(owner, space uuid.UUID) ([]convosRepo.Conversation, error) {
	return r.getSpaceConversations(owner, space)
}

func (r *conversationsRepoStub) GetConversation(owner, space, conversation uuid.UUID) (convosRepo.Conversation, error) {
	return r.getConversation(owner, space, conversation)
}

func (r *conversationsRepoStub) CreateConversation(owner, space uuid.UUID, conversation convosRepo.Conversation) (convosRepo.Conversation, error) {
	return r.createConversation(owner, space, conversation)
}

func (r *conversationsRepoStub) UpdateConversation(owner, space uuid.UUID, conversation convosRepo.Conversation) (convosRepo.Conversation, error) {
	return r.updateConversation(owner, space, conversation)
}

func (r *conversationsRepoStub) DeleteConversation(owner, space, conversation uuid.UUID) error {
	return r.deleteConversation(owner, space, conversation)
}

type messagesRepoStub struct {
	getWholeConversation func(int) ([]messages.MessageWithRoleName, error)
}

func (r *messagesRepoStub) CreateMessage(messages.Message) error { return nil }

func (r *messagesRepoStub) GetRole(string) (messages.MessageRole, error) {
	return messages.MessageRole{}, nil
}

func (r *messagesRepoStub) GetWholeConversation(conversationID int) ([]messages.MessageWithRoleName, error) {
	return r.getWholeConversation(conversationID)
}

func TestGetConversationsMapsRowsAndScopesRepositoryCall(t *testing.T) {
	createdAt := time.Date(2026, time.October, 1, 12, 30, 45, 123000, time.UTC)
	var gotOwner, gotSpace uuid.UUID
	repo := &conversationsRepoStub{
		getSpaceConversations: func(owner, space uuid.UUID) ([]convosRepo.Conversation, error) {
			gotOwner, gotSpace = owner, space
			return []convosRepo.Conversation{{Uuid: testConvoUUID, Title: "A conversation", CreatedAt: createdAt}}, nil
		},
	}
	service := &Service{repo: repo}

	got, err := service.GetConversations(testOwnerUUID, testSpaceUUID)
	if err != nil {
		t.Fatalf("GetConversations returned an error: %v", err)
	}
	if gotOwner.String() != testOwnerUUID || gotSpace.String() != testSpaceUUID {
		t.Fatalf("repository called with owner %s and space %s", gotOwner, gotSpace)
	}
	want := []models.ConversationDto{{Uuid: testConvoUUID, Title: "A conversation", CreatedAt: createdAt}}
	if len(got) != 1 || got[0] != want[0] {
		t.Fatalf("GetConversations() = %#v, want %#v", got, want)
	}
}

func TestGetConversationsReturnsEmptySliceAndMapsRepositoryError(t *testing.T) {
	t.Run("empty result is non-nil", func(t *testing.T) {
		service := &Service{repo: &conversationsRepoStub{
			getSpaceConversations: func(uuid.UUID, uuid.UUID) ([]convosRepo.Conversation, error) {
				return nil, nil
			},
		}}
		got, err := service.GetConversations(testOwnerUUID, testSpaceUUID)
		if err != nil {
			t.Fatalf("GetConversations returned an error: %v", err)
		}
		if got == nil || len(got) != 0 {
			t.Fatalf("expected non-nil empty result, got %#v", got)
		}
	})

	t.Run("repository failure is unexpected error", func(t *testing.T) {
		service := &Service{repo: &conversationsRepoStub{
			getSpaceConversations: func(uuid.UUID, uuid.UUID) ([]convosRepo.Conversation, error) {
				return nil, errors.New("database unavailable")
			},
		}}
		_, err := service.GetConversations(testOwnerUUID, testSpaceUUID)
		assertHandlerError(t, err, http.StatusInternalServerError, "unexpected_error")
	})
}

func TestConversationMethodsRejectInvalidUUIDs(t *testing.T) {
	valid := testOwnerUUID
	tests := []struct {
		name string
		call func(*Service) error
	}{
		{"list owner", func(s *Service) error { _, err := s.GetConversations("invalid", testSpaceUUID); return err }},
		{"list space", func(s *Service) error { _, err := s.GetConversations(valid, "invalid"); return err }},
		{"get owner", func(s *Service) error {
			_, err := s.GetConversation("invalid", testSpaceUUID, testConvoUUID)
			return err
		}},
		{"get space", func(s *Service) error { _, err := s.GetConversation(valid, "invalid", testConvoUUID); return err }},
		{"get conversation", func(s *Service) error { _, err := s.GetConversation(valid, testSpaceUUID, "invalid"); return err }},
		{"update owner", func(s *Service) error {
			_, err := s.UpdateConversation("invalid", testSpaceUUID, testConvoUUID, "title")
			return err
		}},
		{"update space", func(s *Service) error {
			_, err := s.UpdateConversation(valid, "invalid", testConvoUUID, "title")
			return err
		}},
		{"update conversation", func(s *Service) error {
			_, err := s.UpdateConversation(valid, testSpaceUUID, "invalid", "title")
			return err
		}},
		{"delete owner", func(s *Service) error { return s.DeleteConversation("invalid", testSpaceUUID, testConvoUUID) }},
		{"delete space", func(s *Service) error { return s.DeleteConversation(valid, "invalid", testConvoUUID) }},
		{"delete conversation", func(s *Service) error { return s.DeleteConversation(valid, testSpaceUUID, "invalid") }},
		{"full conversation owner", func(s *Service) error {
			_, err := s.GetFullConversation("invalid", testSpaceUUID, testConvoUUID)
			return err
		}},
		{"full conversation space", func(s *Service) error { _, err := s.GetFullConversation(valid, "invalid", testConvoUUID); return err }},
		{"full conversation uuid", func(s *Service) error { _, err := s.GetFullConversation(valid, testSpaceUUID, "invalid"); return err }},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assertHandlerError(t, tt.call(&Service{}), http.StatusBadRequest, "invalid_request")
		})
	}
}

func TestGetConversationMapsNotFound(t *testing.T) {
	service := &Service{repo: &conversationsRepoStub{
		getConversation: func(uuid.UUID, uuid.UUID, uuid.UUID) (convosRepo.Conversation, error) {
			return convosRepo.Conversation{}, &repo_errors.ConversationNotFound{}
		},
	}}
	_, err := service.GetConversation(testOwnerUUID, testSpaceUUID, testConvoUUID)
	assertHandlerError(t, err, http.StatusNotFound, "convo_not_found")
}

func TestUpdateConversationChangesTitleAndReturnsUpdatedDTO(t *testing.T) {
	createdAt := time.Date(2026, time.October, 2, 1, 2, 3, 0, time.UTC)
	initial := convosRepo.Conversation{ID: 7, Uuid: testConvoUUID, Title: "Old", CreatedAt: createdAt}
	var gotOwner, gotSpace uuid.UUID
	var updated convosRepo.Conversation
	repo := &conversationsRepoStub{
		getConversation: func(owner, space, conversation uuid.UUID) (convosRepo.Conversation, error) {
			gotOwner, gotSpace = owner, space
			if conversation.String() != testConvoUUID {
				t.Fatalf("unexpected conversation UUID: %s", conversation)
			}
			return initial, nil
		},
		updateConversation: func(owner, space uuid.UUID, conversation convosRepo.Conversation) (convosRepo.Conversation, error) {
			gotOwner, gotSpace = owner, space
			updated = conversation
			conversation.Title = "New title"
			return conversation, nil
		},
	}
	service := &Service{repo: repo}

	got, err := service.UpdateConversation(testOwnerUUID, testSpaceUUID, testConvoUUID, "New title")
	if err != nil {
		t.Fatalf("UpdateConversation returned an error: %v", err)
	}
	if gotOwner.String() != testOwnerUUID || gotSpace.String() != testSpaceUUID {
		t.Fatalf("repository called with owner %s and space %s", gotOwner, gotSpace)
	}
	if updated.ID != initial.ID || updated.Uuid != initial.Uuid || updated.Title != "New title" {
		t.Fatalf("repository received updated conversation %#v", updated)
	}
	if got.Uuid != testConvoUUID || got.Title != "New title" || !got.CreatedAt.Equal(createdAt) {
		t.Fatalf("UpdateConversation() = %#v", got)
	}
}

func TestDeleteConversationScopesCallAndMapsNotFound(t *testing.T) {
	var gotOwner, gotSpace, gotConversation uuid.UUID
	service := &Service{repo: &conversationsRepoStub{
		deleteConversation: func(owner, space, conversation uuid.UUID) error {
			gotOwner, gotSpace, gotConversation = owner, space, conversation
			return &repo_errors.ConversationNotFound{}
		},
	}}
	err := service.DeleteConversation(testOwnerUUID, testSpaceUUID, testConvoUUID)
	if gotOwner.String() != testOwnerUUID || gotSpace.String() != testSpaceUUID || gotConversation.String() != testConvoUUID {
		t.Fatalf("DeleteConversation passed unexpected UUIDs: %s, %s, %s", gotOwner, gotSpace, gotConversation)
	}
	assertHandlerError(t, err, http.StatusNotFound, "convo_not_found")
}

func TestGetFullConversationMapsMessagesAndUsesPersistedID(t *testing.T) {
	createdAt := time.Date(2026, time.October, 3, 4, 5, 6, 789000, time.UTC)
	var gotID int
	service := &Service{
		repo: &conversationsRepoStub{
			getConversation: func(uuid.UUID, uuid.UUID, uuid.UUID) (convosRepo.Conversation, error) {
				return convosRepo.Conversation{ID: 42, Uuid: testConvoUUID}, nil
			},
		},
		messagesRepo: &messagesRepoStub{
			getWholeConversation: func(conversationID int) ([]messages.MessageWithRoleName, error) {
				gotID = conversationID
				return []messages.MessageWithRoleName{{
					Message:  messages.Message{Uuid: "44444444-4444-4444-8444-444444444444", Content: "hello", CreatedAt: createdAt},
					RoleName: "user",
				}}, nil
			},
		},
	}

	got, err := service.GetFullConversation(testOwnerUUID, testSpaceUUID, testConvoUUID)
	if err != nil {
		t.Fatalf("GetFullConversation returned an error: %v", err)
	}
	if gotID != 42 {
		t.Fatalf("message repository called with conversation ID %d, want 42", gotID)
	}
	if len(got) != 1 || got[0].Uuid != "44444444-4444-4444-8444-444444444444" || got[0].Content != "hello" || got[0].RoleName != "user" || !got[0].CreatedAt.Equal(createdAt) {
		t.Fatalf("GetFullConversation() = %#v", got)
	}
}

func TestGetFullConversationStopsOnNotFoundAndMapsMessageErrors(t *testing.T) {
	t.Run("does not read messages for missing conversation", func(t *testing.T) {
		messageRead := false
		service := &Service{
			repo: &conversationsRepoStub{
				getConversation: func(uuid.UUID, uuid.UUID, uuid.UUID) (convosRepo.Conversation, error) {
					return convosRepo.Conversation{}, &repo_errors.ConversationNotFound{}
				},
			},
			messagesRepo: &messagesRepoStub{
				getWholeConversation: func(int) ([]messages.MessageWithRoleName, error) {
					messageRead = true
					return nil, nil
				},
			},
		}
		_, err := service.GetFullConversation(testOwnerUUID, testSpaceUUID, testConvoUUID)
		assertHandlerError(t, err, http.StatusNotFound, "convo_not_found")
		if messageRead {
			t.Fatal("messages repository called for a missing conversation")
		}
	})

	t.Run("message repository failure is unexpected error", func(t *testing.T) {
		service := &Service{
			repo: &conversationsRepoStub{
				getConversation: func(uuid.UUID, uuid.UUID, uuid.UUID) (convosRepo.Conversation, error) {
					return convosRepo.Conversation{ID: 42}, nil
				},
			},
			messagesRepo: &messagesRepoStub{
				getWholeConversation: func(int) ([]messages.MessageWithRoleName, error) {
					return nil, errors.New("database unavailable")
				},
			},
		}
		_, err := service.GetFullConversation(testOwnerUUID, testSpaceUUID, testConvoUUID)
		assertHandlerError(t, err, http.StatusInternalServerError, "unexpected_error")
	})
}

func TestFullConversationToDtoReturnsEmptySlice(t *testing.T) {
	dto := fullConversationToDto(nil)
	if dto == nil || len(dto) != 0 {
		t.Fatalf("expected non-nil empty DTO slice, got %#v", dto)
	}
}

func assertHandlerError(t *testing.T, err error, wantStatus int, wantCode string) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected error %q", wantCode)
	}
	recorder := httptest.NewRecorder()
	handler_errors.WriteError(recorder, err)
	if recorder.Code != wantStatus {
		t.Fatalf("expected status %d, got %d", wantStatus, recorder.Code)
	}
	var got handler_errors.AppError
	if decodeErr := json.NewDecoder(recorder.Body).Decode(&got); decodeErr != nil {
		t.Fatalf("decode error response: %v", decodeErr)
	}
	if got.Code != wantCode {
		t.Fatalf("expected error code %q, got %q", wantCode, got.Code)
	}
}
