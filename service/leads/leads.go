package leads

import (
	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/jinzhu/gorm"
	intercom "gopkg.in/intercom/intercom-go.v2"
	"log"
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
	return &leads{
		intercom: intercom.NewClient(config.AccessToken, ""),
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

	var tag *intercom.Tag

	for _, t := range tags.Tags {
		if t.Name == "invite_ready" {
			tag = &t
			break
		}
	}

	log.Printf("Searching contacts")
	contacts, err := s.intercom.Contacts.ListByTag("can_invite", intercom.PageParams{PerPage: int64(num), TotalPages: 1})
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

		newTags := []intercom.Tag{*tag}
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
