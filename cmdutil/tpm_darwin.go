//go:build darwin
// +build darwin

package cmdutil

import (
	"encoding/json"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/keybase/go-keychain"
	"github.com/src-bin/substrate/naming"
	"github.com/src-bin/substrate/ui"
)

func SetTPM(creds aws.Credentials) error {
	ui.Printf("storing access key %s (expires %s) in the macOS keychain", creds.AccessKeyID, creds.Expires.Format(time.RFC3339))
	item := keychain.NewItem()
	item.SetAccessible(keychain.AccessibleWhenUnlocked)
	item.SetSecClass(keychain.SecClassGenericPassword)
	item.SetSynchronizable(keychain.SynchronizableNo)

	prefix, err := naming.PrefixNoninteractive()
	if err != nil {
		return err
	}
	item.SetAccount(prefix)

	data, err := json.Marshal(creds)
	if err != nil {
		return err
	}
	item.SetData(data)

	item.SetService(naming.Substrate)

	err = keychain.AddItem(item)
	if err == keychain.ErrorDuplicateItem {
		err = keychain.UpdateItem(item, item)
	}
	return err
}

func SetenvFromTPM() error {
	query := keychain.NewItem()
	query.SetMatchLimit(keychain.MatchLimitOne)
	query.SetReturnAttributes(true)
	query.SetReturnData(true)
	query.SetSecClass(keychain.SecClassGenericPassword)

	prefix, err := naming.PrefixNoninteractive()
	if err != nil {
		return err
	}
	query.SetAccount(prefix)

	query.SetService(naming.Substrate)

	results, err := keychain.QueryItem(query)
	if err != nil {
		return err
	}
	for _, result := range results {
		if result.Account == prefix && result.Service == naming.Substrate { // belt and suspenders

			var creds aws.Credentials
			if err := json.Unmarshal(result.Data, &creds); err != nil {
				return err
			}
			if err := Setenv(creds); err != nil {
				return err
			}
			ui.Printf("found access key %s (expires %s) in the macOS keychain", creds.AccessKeyID, creds.Expires.Format(time.RFC3339))

		}
	}
	return nil // or should we concoct an error here?
}
