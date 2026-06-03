// SPDX-License-Identifier: GPL-2.0-or-later

package validations

import "fmt"

type gnssFetchError struct {
	id          string
	description string
	order       int
	err         error
}

func (g *gnssFetchError) Verify() error {
	return fmt.Errorf("GNSS data unavailable: %w", g.err)
}

func (g *gnssFetchError) GetID() string {
	return g.id
}

func (g *gnssFetchError) GetDescription() string {
	return g.description
}

func (g *gnssFetchError) GetData() any { //nolint:ireturn // data will vary for each validation
	return map[string]string{"error": g.err.Error()}
}

func (g *gnssFetchError) GetOrder() int {
	return g.order
}

func NewUnknownGNSSAntStatus(err error) Validation {
	return &gnssFetchError{
		id:          gnssAntStatusID,
		description: gnssAntStatusDescription,
		order:       gnssConnectedToAntOrdering,
		err:         err,
	}
}

func NewUnknownGNSSNavStatus(err error) Validation {
	return &gnssFetchError{
		id:          gnssStatusID,
		description: gnssStatusDescription,
		order:       gnssReceivingDataOrdering,
		err:         err,
	}
}
