// Copyright (c) HashiCorp, Inc.
// SPDX-License-Identifier: MPL-2.0

package mssql

import (
	"fmt"
	"log"
	"time"

	"github.com/hashicorp/go-azure-helpers/lang/pointer"
	"github.com/hashicorp/go-azure-helpers/lang/response"
	"github.com/hashicorp/go-azure-sdk/resource-manager/sql/2023-08-01-preview/jobcredentials"
	"github.com/hashicorp/go-cty/cty"
	"github.com/hashicorp/terraform-provider-azurerm/helpers/tf"
	"github.com/hashicorp/terraform-provider-azurerm/internal/clients"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/mssql/parse"
	"github.com/hashicorp/terraform-provider-azurerm/internal/services/mssql/validate"
	"github.com/hashicorp/terraform-provider-azurerm/internal/tf/pluginsdk"
	"github.com/hashicorp/terraform-provider-azurerm/internal/timeouts"
)

func resourceMsSqlJobCredential() *pluginsdk.Resource {
	return &pluginsdk.Resource{
		Create: resourceMsSqlJobCredentialCreate,
		Read:   resourceMsSqlJobCredentialRead,
		Update: resourceMsSqlJobCredentialUpdate,
		Delete: resourceMsSqlJobCredentialDelete,

		Importer: pluginsdk.ImporterValidatingResourceId(func(id string) error {
			_, err := parse.JobCredentialID(id)
			return err
		}),

		Timeouts: &pluginsdk.ResourceTimeout{
			Create: pluginsdk.DefaultTimeout(60 * time.Minute),
			Read:   pluginsdk.DefaultTimeout(5 * time.Minute),
			Update: pluginsdk.DefaultTimeout(60 * time.Minute),
			Delete: pluginsdk.DefaultTimeout(60 * time.Minute),
		},

		Schema: map[string]*pluginsdk.Schema{
			"name": {
				Type:     pluginsdk.TypeString,
				Required: true,
				ForceNew: true,
			},

			"job_agent_id": {
				Type:         pluginsdk.TypeString,
				Required:     true,
				ForceNew:     true,
				ValidateFunc: validate.JobAgentID,
			},

			"username": {
				Type:     pluginsdk.TypeString,
				Required: true,
			},

			"password": {
				Type:          pluginsdk.TypeString,
				Optional:      true,
				Sensitive:     true,
				ConflictsWith: []string{"password_wo"},
				ExactlyOneOf:  []string{"password", "password_wo"},
			},
			"password_wo": {
				Type:          pluginsdk.TypeString,
				Optional:      true,
				WriteOnly:     true,
				RequiredWith:  []string{"password_wo_version"},
				ConflictsWith: []string{"password"},
				ExactlyOneOf:  []string{"password_wo", "password"},
			},
			"password_wo_version": {
				Type:         pluginsdk.TypeInt,
				Optional:     true,
				RequiredWith: []string{"password_wo"},
			},
		},
	}
}

func resourceMsSqlJobCredentialCreate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MSSQL.JobCredentialsClient
	ctx, cancel := timeouts.ForCreate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for Job Credential creation.")

	jaId, err := jobcredentials.ParseJobAgentID(d.Get("job_agent_id").(string))
	if err != nil {
		return err
	}
	jobCredentialId := jobcredentials.NewCredentialID(jaId.SubscriptionId, jaId.ResourceGroupName, jaId.ServerName, jaId.JobAgentName, d.Get("name").(string))

	existing, err := client.Get(ctx, jobCredentialId)
	if err != nil && !response.WasNotFound(existing.HttpResponse) {
		return fmt.Errorf("checking for presence of existing %s: %+v", jobCredentialId, err)
	}

	if !response.WasNotFound(existing.HttpResponse) {
		return tf.ImportAsExistsError("azurerm_mssql_job_credential", jobCredentialId.ID())
	}

	woPassword, err := pluginsdk.GetWriteOnly(d, "password_wo", cty.String)
	if err != nil {
		return err
	}

	password := d.Get("password").(string)
	if !woPassword.IsNull() {
		password = woPassword.AsString()
	}

	jobCredential := jobcredentials.JobCredential{
		Name: pointer.To(jobCredentialId.CredentialName),
		Properties: &jobcredentials.JobCredentialProperties{
			Username: d.Get("username").(string),
			Password: password,
		},
	}

	if _, err := client.CreateOrUpdate(ctx, jobCredentialId, jobCredential); err != nil {
		return fmt.Errorf("creating %s: %+v", jobCredentialId, err)
	}

	d.SetId(jobCredentialId.ID())

	return resourceMsSqlJobCredentialRead(d, meta)
}

func resourceMsSqlJobCredentialUpdate(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MSSQL.JobCredentialsClient
	ctx, cancel := timeouts.ForUpdate(meta.(*clients.Client).StopContext, d)
	defer cancel()

	log.Printf("[INFO] preparing arguments for Job Credential update.")

	jaId, err := jobcredentials.ParseJobAgentID(d.Get("job_agent_id").(string))
	if err != nil {
		return err
	}
	jobCredentialId := jobcredentials.NewCredentialID(jaId.SubscriptionId, jaId.ResourceGroupName, jaId.ServerName, jaId.JobAgentName, d.Get("name").(string))

	existing, err := client.Get(ctx, jobCredentialId)
	if err != nil {
		return fmt.Errorf("retrieving %s: %+v", jobCredentialId, err)
	}

	if existing.Model == nil {
		return fmt.Errorf("retrieving %s: `model` was nil", jobCredentialId)
	}

	if existing.Model.Properties == nil {
		return fmt.Errorf("retrieving %s: `model.Properties` was nil", jobCredentialId)
	}
	payload := existing.Model

	if d.HasChange("username") {
		payload.Properties.Username = d.Get("username").(string)
	}

	if d.HasChange("password") {
		payload.Properties.Password = d.Get("password").(string)
	}

	if d.HasChange("password_wo_version") {
		woPassword, err := pluginsdk.GetWriteOnly(d, "password_wo", cty.String)
		if err != nil {
			return err
		}

		if !woPassword.IsNull() {
			payload.Properties.Password = woPassword.AsString()
		}
	}

	if _, err := client.CreateOrUpdate(ctx, jobCredentialId, *payload); err != nil {
		return fmt.Errorf("updating %s: %+v", jobCredentialId, err)
	}

	return resourceMsSqlJobCredentialRead(d, meta)
}

func resourceMsSqlJobCredentialRead(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MSSQL.JobCredentialsClient
	ctx, cancel := timeouts.ForRead(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := jobcredentials.ParseCredentialID(d.Id())
	if err != nil {
		return err
	}

	resp, err := client.Get(ctx, *id)
	if err != nil {
		if response.WasNotFound(resp.HttpResponse) {
			d.SetId("")
			return nil
		}
		return fmt.Errorf("reading %s: %s", *id, err)
	}

	d.Set("name", id.CredentialName)
	jobAgentId := jobcredentials.NewJobAgentID(id.SubscriptionId, id.ResourceGroupName, id.ServerName, id.JobAgentName)
	d.Set("job_agent_id", jobAgentId.ID())

	if model := resp.Model; model != nil {
		if props := model.Properties; props != nil {
			d.Set("username", props.Username)
		}
	}

	d.Set("password_wo_version", d.Get("password_wo_version").(int))

	return nil
}

func resourceMsSqlJobCredentialDelete(d *pluginsdk.ResourceData, meta interface{}) error {
	client := meta.(*clients.Client).MSSQL.JobCredentialsClient
	ctx, cancel := timeouts.ForDelete(meta.(*clients.Client).StopContext, d)
	defer cancel()

	id, err := jobcredentials.ParseCredentialID(d.Id())
	if err != nil {
		return err
	}

	_, err = client.Delete(ctx, *id)
	if err != nil {
		return fmt.Errorf("deleting %s: %+v", id, err)
	}

	return nil
}
