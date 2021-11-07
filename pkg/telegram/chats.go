package telegram

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/docker/libkv/store"
	"github.com/tucnak/telebot"
)

const telegramChatsDirectory = "telegram/chats"

// ChatStore writes the users to a libkv store backend
type ChatStore struct {
	kv store.Store
}

// AugmentedChat - telebot.Chat with user options to filter alerts by labels
type AugmentedChat struct {
	UserLabelFilters map[string]string
	telebot.Chat
}

// NewChatStore stores telegram chats in the provided kv backend
func NewChatStore(kv store.Store) (*ChatStore, error) {
	return &ChatStore{kv: kv}, nil
}

// NewAugmentedChat parse telebot.Message.Text field to get label filters
func NewAugmentedChat(message telebot.Message) AugmentedChat {
	// First field is the command, like '/start', just skip it
	payload := strings.Fields(message.Text)[1:]
	userLabelFilters := make(map[string]string)
	for _, field := range payload {
		data := strings.SplitN(field, "=", 2)
		if len(data) == 2 {
			userLabelFilters[data[0]] = data[1]
		}
	}
	return AugmentedChat{userLabelFilters, message.Chat}
}

// CheckFilters - compare filters against labels and return true if filters passed
func (c *AugmentedChat) CheckFilters(labels map[string]string) bool {
	for key, value := range c.UserLabelFilters {
		if label, ok := labels[key]; !ok || label != value {
			fmt.Println("Failed filter: ", label, value)
			return false
		}
	}
	return true
}

// List all chats saved in the kv backend
func (s *ChatStore) List() ([]AugmentedChat, error) {
	kvPairs, err := s.kv.List(telegramChatsDirectory)
	if err != nil {
		return nil, err
	}

	var chats []AugmentedChat
	for _, kv := range kvPairs {
		var c AugmentedChat
		if err := json.Unmarshal(kv.Value, &c); err != nil {
			return nil, err
		}
		chats = append(chats, c)
	}

	return chats, nil
}

// Add a telegram chat to the kv backend
func (s *ChatStore) Add(c AugmentedChat) error {
	b, err := json.Marshal(c)
	if err != nil {
		return err
	}

	key := fmt.Sprintf("%s/%d", telegramChatsDirectory, c.ID)

	return s.kv.Put(key, b, nil)
}

// Remove a telegram chat from the kv backend
func (s *ChatStore) Remove(c AugmentedChat) error {
	key := fmt.Sprintf("%s/%d", telegramChatsDirectory, c.ID)
	return s.kv.Delete(key)
}
