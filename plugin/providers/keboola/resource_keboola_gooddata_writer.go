package keboola

import (
	"bytes"
	"encoding/json"
	"log"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/hashicorp/terraform/helper/schema"
)

type CreateGoodDataProject struct {
	WriterID    string `json:"writerId"`
	Description string `json:"description"`
	AuthToken   string `json:"authToken"`
}

type GoodDataWriter struct {
	ID          string `json:"id,omitempty"`
	Name        string `json:"name"`
	Description string `json:"description"`
	AuthToken   string `json:"authToken"`
}

func resourceKeboolaGoodDataWriter() *schema.Resource {
	return &schema.Resource{
		Create: resourceKeboolaGoodDataWriterCreate,
		Read:   resourceKeboolaGoodDataWriterRead,
		Update: resourceKeboolaGoodDataWriterUpdate,
		Delete: resourceKeboolaGoodDataWriterDelete,

		Schema: map[string]*schema.Schema{
			"name": &schema.Schema{
				Type:     schema.TypeString,
				Required: true,
			},
			"description": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
			},
			"authToken": &schema.Schema{
				Type:     schema.TypeString,
				Optional: true,
				Default:  "keboola_demo",
			},
		},
	}
}

func resourceKeboolaGoodDataWriterCreate(d *schema.ResourceData, meta interface{}) error {
	log.Print("[INFO] Creating GoodData Writer in Keboola.")

	name := d.Get("name").(string)
	reg, err := regexp.Compile("[^A-Za-z0-9]+")
	if err != nil {
		return err
	}

	generatedID := reg.ReplaceAllString(name, "-")
	generatedID = strings.ToLower(strings.Trim(generatedID, "-"))

	createProject := CreateGoodDataProject{
		WriterID:    generatedID,
		Description: d.Get("description").(string),
		AuthToken:   d.Get("authToken").(string),
	}

	createJSON, err := json.Marshal(createProject)
	if err != nil {
		return err
	}

	client := meta.(*KbcClient)

	createBuffer := bytes.NewBuffer(createJSON)
	createWriterResp, err := client.PostToSyrup("gooddata-writer/v2", createBuffer)

	if hasErrors(err, createWriterResp) {
		return extractError(err, createWriterResp)
	}

	createWriterStatus := "waiting"
	var createWriterStatusRes StorageJobStatus

	createWriterDecoder := json.NewDecoder(createWriterResp.Body)
	err = createWriterDecoder.Decode(&createWriterStatusRes)

	if err != nil {
		return err
	}

	jobURL, err := url.Parse(createWriterStatusRes.URL)

	if err != nil {
		return err
	}

	for createWriterStatus != "success" && createWriterStatus != "error" {
		jobStatusResp, err := client.GetFromSyrup(strings.TrimLeft(jobURL.Path, "/"))

		if hasErrors(err, jobStatusResp) {
			return extractError(err, jobStatusResp)
		}

		decoder := json.NewDecoder(jobStatusResp.Body)
		err = decoder.Decode(&createWriterStatusRes)

		if err != nil {
			return err
		}

		time.Sleep(250 * time.Millisecond)
		createWriterStatus = createWriterStatusRes.Status
	}

	form := url.Values{}
	form.Add("name", d.Get("name").(string))
	form.Add("description", d.Get("description").(string))
	form.Add("configurationId", generatedID)

	formdataBuffer := bytes.NewBufferString(form.Encode())

	createWriterConfigResp, err := client.PostToStorage("v2/storage/components/gooddata-writer/configs", formdataBuffer)

	if err != nil {
		return err
	}

	if hasErrors(err, createWriterConfigResp) {
		return extractError(err, createWriterConfigResp)
	}

	var createRes CreateResourceResult

	createDecoder := json.NewDecoder(createWriterConfigResp.Body)
	err = createDecoder.Decode(&createRes)

	if err != nil {
		return err
	}

	d.SetId(string(createRes.ID))

	return resourceKeboolaGoodDataWriterRead(d, meta)
}

func resourceKeboolaGoodDataWriterRead(d *schema.ResourceData, meta interface{}) error {
	log.Print("[INFO] Reading Access Tokens from Keboola.")

	return nil
}

func resourceKeboolaGoodDataWriterUpdate(d *schema.ResourceData, meta interface{}) error {
	log.Print("[INFO] Updating Access Token in Keboola.")

	return resourceKeboolaGoodDataWriterRead(d, meta)
}

func resourceKeboolaGoodDataWriterDelete(d *schema.ResourceData, meta interface{}) error {
	log.Printf("[INFO] Deleting Access Token in Keboola: %s", d.Id())

	return nil
}
