package whatsmeow

import (
	"context"
	"fmt"

	"go.mau.fi/whatsmeow/appstate"
	"go.mau.fi/whatsmeow/types"
)

// SaveContact sends an app state patch to save or update a contact on the WhatsApp server.
// On success, the server applies the contact mutation, app state is resynced, and the
// local ContactStore is updated via the existing app state handling logic.
func (cli *Client) SaveContact(
	ctx context.Context,
	jid types.JID,
	firstName, fullName string,
	opts appstate.ContactOptions,
) error {
	if cli == nil {
		return ErrClientIsNil
	}
	if jid.IsEmpty() {
		return fmt.Errorf("SaveContact: empty JID")
	}

	jid = jid.ToNonAD()

	patch := appstate.BuildContact(jid, firstName, fullName, opts)
	return cli.SendAppState(ctx, patch)
}

