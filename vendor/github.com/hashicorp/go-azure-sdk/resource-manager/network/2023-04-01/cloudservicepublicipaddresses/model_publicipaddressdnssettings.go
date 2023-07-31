package cloudservicepublicipaddresses

// Copyright (c) Microsoft Corporation. All rights reserved.
// Licensed under the MIT License. See NOTICE.txt in the project root for license information.

type PublicIPAddressDnsSettings struct {
	DomainNameLabel      *string                                         `json:"domainNameLabel,omitempty"`
	DomainNameLabelScope *PublicIPAddressDnsSettingsDomainNameLabelScope `json:"domainNameLabelScope,omitempty"`
	Fqdn                 *string                                         `json:"fqdn,omitempty"`
	ReverseFqdn          *string                                         `json:"reverseFqdn,omitempty"`
}