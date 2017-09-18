package leads

import (
	"errors"
	"log"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/jinzhu/gorm"
	intercom "gopkg.in/intercom/intercom-go.v2"
)

/*
Leads encapsulates the logic for inviting, and converting leads
*/

type Leads interface {
	// For each user with a can_invite tag, create an invite token for them, and add it to Intercom
	// return number of invited, or an error
	Invite(num int) (int, error)

	// If a user signs up with an InviteToken, convert from contact to a user
	Invited(token models.InviteToken, user models.User) error
}

type leads struct {
	intercom *intercom.Client
	db       *gorm.DB
}

func New(config events.IntercomConfig, db *gorm.DB) Leads {
	c := intercom.NewClient(config.AccessToken, "")
	return &leads{
		intercom: c,
		db:       db,
	}
}

func (s *leads) Invite(num int) (invited int, err error) {
	invited = 0

	log.Printf("Searching tags")
	tags, err := s.intercom.Tags.List()
	if err != nil {
		return
	}

	var readyTag intercom.Tag
	var canInviteTag intercom.Tag

	for _, t := range tags.Tags {

		if t.Name == "invite_ready" {
			readyTag = t
		}

		if t.Name == "can_invite" {
			canInviteTag = t
		}

	}

	if readyTag.ID == "" {
		return 0, errors.New("Can't find a tag 'invite_ready'")
	}

	if canInviteTag.ID == "" {
		return 0, errors.New("Can't find a tag 'can_invite'")
	}

	contacts, err := s.intercom.Contacts.ListByTag(canInviteTag.ID, intercom.PageParams{})
	if err != nil {
		return
	}

	for _, c := range contacts.Contacts {
		t := models.NewInviteToken()
		t.IntercomId = c.UserID
		err = s.db.Create(&t).Error
		if err != nil {
			return
		}

		newTags := []intercom.Tag{readyTag}
		c.Tags = &(intercom.TagList{Tags: newTags})
		c.CustomAttributes["invite_token"] = t.Token
		s.intercom.Contacts.Update(&c)
		if err != nil {
			return
		}
		invited += 1
	}
	return
}

func (s *leads) Invited(token models.InviteToken, user models.User) error {
	contact := intercom.Contact{UserID: token.IntercomId}
	intercom_user := intercom.User{Email: user.Email}
	_, err := s.intercom.Contacts.Convert(&contact, &intercom_user)
	return err
}
