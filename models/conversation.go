package models

import (
	u "go-chat/utils"

	uuid "github.com/satori/go.uuid"
)

// Conversation is what groups together users into a single conversation thread
type Conversation struct {
	BaseFields
	Messages []*Message `json:"messages"`
	Users    []*User    `gorm:"many2many:user_conversation_join;" json:"users"`
	// association_foreignkey:userId;foreignkey:conversationId
}

// Create creates or looks up a new conversation that has all of the passed in users
func (c *Conversation) Create(usrIds []string) map[string]interface{} {

	// TODO: Add code to check if conversation between certain users already exists

	// Generate and set ID field using uuid v4
	id, err := uuid.NewV4()
	if err != nil {
		return u.Message(false, "Failed to create account, error creating ID")
	}
	c.ID = id

	var users []*User
	// var id uuid.UUID

	for _, usrID := range usrIds {
		id := u.UUIDFromString(usrID)
		user := GetUserByID(id)
		users = append(users, user)
	}

	c.Users = users

	db := GetDB()
	db.Debug().Create(&c)

	// Compose a response
	response := u.Message(false, "conversation has been created")
	// Attach the user to the response
	response["conversation"] = c

	return response
}

// GetMessagesByConversationID returns the messages from a conversation
func (c *Conversation) GetMessagesByConversationID() error {
	var err error
	// Lookup conversations related messages
	db := GetDB()
	err = db.Where("conversation_id = ?", c.ID).Find(&c.Messages).Error
	if err != nil {
		return err
	}
	// Attach a user to each message
	for _, message := range c.Messages {
		err = message.GetMessageUser()
		if err != nil {
			return err
		}
	}
	return nil
}

// GetConversations returns all conversations from the db
func GetConversations() map[string]interface{} {

	var conv []*Conversation

	db := GetDB()
	convs := db.Find(&conv)

	// Compose response
	resp := u.Message(false, "conversations retreived")
	// Attatch conversations to the response
	resp["conversations"] = convs

	return resp
}

// GetConversationByID gets a single conversation by id
func GetConversationByID(id uuid.UUID) Conversation {

	c := Conversation{}
	db := GetDB()
	db.Where("id = ?", id).Find(&c)

	return c
}

// GetConversationsByUserID Returns a user struct with all of their conversations attatched to it
func GetConversationsByUserID(id uuid.UUID) map[string]interface{} {
	// Get db entrypoint
	db := GetDB()

	// Get the current User and their associated Conversations
	usr := User{}
	db.Preload("Conversations").Where("id = ?", id).First(&usr)

	// Look up The Users associated with each conversation
	for _, conv := range usr.Conversations {
		// Create a place to push temporary copy of users slice
		tempUsers := []*User{}
		// Get all of the users associated with a conversation
		db.Debug().Preload("Users").Where("id = ?", conv.ID).First(&conv)
		// Copy only the fields we want to a new user
		for _, u := range conv.Users {
			// Ignore the user requesting the conversation
			if u.ID != id {
				tempUser := *new(User)
				tempUser.ID = u.ID
				tempUser.FirstName = u.FirstName
				tempUser.LastName = u.LastName
				tempUser.Email = u.Email
				tempUsers = append(tempUsers, &tempUser)
			}
		}
		// Reasign Conversation.Users to the copy that was just created
		conv.Users = tempUsers
		// Clear tempUsers slice for next iteration
		tempUsers = []*User{}
	}

	// TODO: Add error handling around passing in non-existent or invalid UserId

	resp := u.Message(false, "conversations retreived")

	// Return an emtpy slice if this value empty
	if usr.Conversations == nil {
		usr.Conversations = make([]*Conversation, 0)
	}

	resp["conversations"] = usr.Conversations

	return resp
}
