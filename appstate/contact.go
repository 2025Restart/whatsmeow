package appstate

import (
	"go.mau.fi/whatsmeow/proto/waSyncAction"
	"go.mau.fi/whatsmeow/types"
)

// ContactOptions contains optional fields when building a contact app state patch.
// This keeps the BuildContact API stable if WhatsApp adds more fields later.
type ContactOptions struct {
	// LID is the LID JID corresponding to the contact, if known.
	LID *types.JID

	// SaveOnPrimaryAddressbook mirrors the ContactAction.saveOnPrimaryAddressbook flag.
	// When nil, the flag is omitted and defaults to false.
	SaveOnPrimaryAddressbook *bool

	// PnJID is reserved for phone-number based JID if needed in the future.
	PnJID *types.JID

	// Username is reserved for future username-based contacts.
	Username *string
}

// contactMutationVersion is the version used for contact mutations.
// This should match the version used by official clients; if that ever changes,
// this constant can be updated based on observed incoming mutations.
const contactMutationVersion int32 = 1

// BuildContact constructs a PatchInfo that updates the contact info for the given JID.
// The resulting patch uses the critical_unblock_low collection with an IndexContact entry,
// matching how incoming contact mutations are represented.
func BuildContact(target types.JID, firstName, fullName string, opts ContactOptions) PatchInfo {
	target = target.ToNonAD()

	index := []string{IndexContact, target.String()}

	contact := &waSyncAction.ContactAction{}
	if fullName != "" {
		contact.FullName = &fullName
	}
	if firstName != "" {
		contact.FirstName = &firstName
	}
	if opts.LID != nil {
		lidStr := opts.LID.ToNonAD().String()
		if lidStr != "" {
			contact.LidJID = &lidStr
		}
	}
	if opts.SaveOnPrimaryAddressbook != nil {
		contact.SaveOnPrimaryAddressbook = opts.SaveOnPrimaryAddressbook
	}

	mutation := MutationInfo{
		Index:   index,
		Version: contactMutationVersion,
		Value: &waSyncAction.SyncActionValue{
			ContactAction: contact,
		},
	}

	return PatchInfo{
		Type:      WAPatchCriticalUnblockLow,
		Mutations: []MutationInfo{mutation},
	}
}

