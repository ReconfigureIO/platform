package leads

import (
	"errors"

	"github.com/ReconfigureIO/platform/models"
	"github.com/ReconfigureIO/platform/service/events"
	"github.com/jinzhu/gorm"
	log "github.com/sirupsen/logrus"
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

	// Updates an intercom customer to match our User
	SyncIntercomCustomer(user models.User) error
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

	// We should be using the contact service, but that doesn't filter by tag
	// Instead, Users will return Contacts, so we'll use that
	contacts, err := s.intercom.Users.ListByTag(canInviteTag.ID, intercom.PageParams{PerPage: int64(num)})
	if err != nil {
		return
	}

	untags := []intercom.Tagging{}
	toTags := []intercom.Tagging{}

	for _, c := range contacts.Users {
		log.Printf("Inviting %v\n", c)

		// create invite
		t := models.NewInviteToken()
		t.IntercomId = c.UserID
		err = s.db.Create(&t).Error
		if err != nil {
			return
		}

		// Get the equivalent contact, since using the user api causes an error
		contact := intercom.Contact{
			ID:                     c.ID,
			Email:                  c.Email,
			Phone:                  c.Phone,
			UserID:                 c.UserID,
			Name:                   c.Name,
			Avatar:                 c.Avatar,
			LocationData:           c.LocationData,
			LastRequestAt:          c.LastRequestAt,
			CreatedAt:              c.CreatedAt,
			UpdatedAt:              c.UpdatedAt,
			SessionCount:           c.SessionCount,
			LastSeenIP:             c.LastSeenIP,
			SocialProfiles:         c.SocialProfiles,
			UnsubscribedFromEmails: c.UnsubscribedFromEmails,
			UserAgentData:          c.UserAgentData,
			Tags:                   c.Tags,
			Segments:               c.Segments,
			Companies:              c.Companies,
			CustomAttributes:       c.CustomAttributes,
			UpdateLastRequestAt:    c.UpdateLastRequestAt,
			NewSession:             c.NewSession,
		}

		// add invite token & tag as `invite_ready`
		contact.CustomAttributes = map[string]interface{}{"invite_token": t.Token}

		log.Printf("Updating Contact %v\n", contact)
		_, err = s.intercom.Contacts.Update(&contact)
		if err != nil {
			return
		}

		// untag `can_invite` so we don't do it again
		untags = append(untags, intercom.Tagging{Untag: intercom.Bool(true), ID: c.ID})

		// tag 'invite_ready` so we don't do it again
		toTags = append(toTags, intercom.Tagging{ID: c.ID})

		invited += 1
	}

	log.Printf("Untagging\n")
	// untag can_invite
	_, err = s.intercom.Tags.Tag(&intercom.TaggingList{
		Name:  "can_invite",
		Users: untags,
	})
	if err != nil {
		return
	}

	log.Printf("Tagging\n")
	// tag invite_ready
	_, err = s.intercom.Tags.Tag(&intercom.TaggingList{
		Name:  "invite_ready",
		Users: toTags,
	})

	return
}

func (s *leads) Invited(token models.InviteToken, user models.User) error {
	if token.IntercomId == "" {
		return nil
	}
	contact := intercom.Contact{UserID: token.IntercomId}
	intercom_user := intercom.User{Email: user.Email}
	_, err := s.intercom.Contacts.Convert(&contact, &intercom_user)
	return err
}

func (s *leads) SyncIntercomCustomer(user models.User) error {
	ic := s.intercom
	icUser, err := ic.Users.FindByUserID(user.ID)
	if err != nil {
		return err
	}

	icUser.Name = user.Name
	icUser.Email = user.Email
	icUser.Phone = user.PhoneNumber
	icUser.CustomAttributes["landing"] = truncateString(user.Landing, 252)
	icUser.CustomAttributes["main_goal"] = truncateString(user.MainGoal, 252)
	icUser.CustomAttributes["employees"] = truncateString(user.Employees, 252)
	icUser.CustomAttributes["market_verticals"] = truncateString(user.MarketVerticals, 252)

	companyList := intercom.CompanyList{
		Companies: []intercom.Company{
			{
				CompanyID: user.ID,
				Name:      truncateString(user.Company, 252),
			},
		},
	}

	icUser.Companies = &companyList

	_, err = ic.Users.Save(&icUser)
	if err != nil {
		return err
	}
	return nil
}

func truncateString(str string, num int) string {
	output := str
	if len(str) > num {
		output = str[0 : num-1]
	}
	return output
}
