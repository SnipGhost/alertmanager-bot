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
	UserLabelFilters map[string]map[string]struct{}
	telebot.Chat
}

// NewAugmentedChat parse telebot.Message.Text field to get label filters
func NewAugmentedChat(message telebot.Message) AugmentedChat {
	// First field is the command, like '/start', just skip it
	payload := strings.Fields(message.Text)[1:]
	userLabelFilters := make(map[string]map[string]struct{})
	for _, field := range payload {
		data := strings.Split(field, "=")
		if len(data) > 1 {
			set := make(map[string]struct{})
			for _, value := range data[1:] {
				set[value] = struct{}{}
			}
			userLabelFilters[data[0]] = set
		}
	}
	return AugmentedChat{userLabelFilters, message.Chat}
}

func (c *AugmentedChat) GetFiltersAsString() string {
	if len(c.UserLabelFilters) == 0 {
		return "Allowed ALL"
	}
	result := make([]string, 0, len(c.UserLabelFilters))
	for key, set := range c.UserLabelFilters {
		values := make([]string, 0, len(set))
		for val := range set {
			values = append(values, val)
		}
		result = append(result, key+"=("+strings.Join(values, "|")+")")
	}
	return strings.Join(result, " & ")
}

// CheckFilters - compare filters against labels and return true if filters passed
func (c *AugmentedChat) CheckFilters(labels map[string]string) bool {
	for key, set := range c.UserLabelFilters {
		label, contains_label := labels[key]
		if !contains_label {
			// Allow omit label with underscore
			if _, allow_omitted := set["_"]; !allow_omitted {
				return false
			}
			continue
		}
		// Deny alerts label with prohibited values (!val)
		if _, deny_value := set["!"+label]; deny_value {
			return false
		}
		// Allow any with asterisk
		if _, allow_any_contained := set["*"]; allow_any_contained {
			continue
		}
		// Set of label values in filters is not contain expected value
		if _, ok := set[label]; !ok {
			return false
		}
	}
	return true
}

// NewChatStore stores telegram chats in the provided kv backend
func NewChatStore(kv store.Store) (*ChatStore, error) {
	return &ChatStore{kv: kv}, nil
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
