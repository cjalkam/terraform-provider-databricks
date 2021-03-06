package access

// Preview feature: https://docs.databricks.com/security/network/ip-access-list.html
// REST API: https://docs.databricks.com/dev-tools/api/latest/ip-access-list.html#operation/create-list

import (
	"log"

	"github.com/databrickslabs/databricks-terraform/common"
	"github.com/databrickslabs/databricks-terraform/internal"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/schema"
	"github.com/hashicorp/terraform-plugin-sdk/v2/helper/validation"
)

// List: Get all IP access lists
// Norequest needed--just a get
type ListIPAccessListsResponse struct {
	ListIPAccessListsResponse []IPAccessListStatus `json:"ip_access_lists,omitempty"`
}

// Update an IP access list
type IPAccessListUpdateRequest struct {
	Label       string           `json:"label,omitempty"`
	ListType    IPAccessListType `json:"list_type,omitempty"`
	IPAddresses []string         `json:"ip_addresses,omitempty"`
	Enabled     bool             `json:"enabled,omitempty"`
}
type IPAccessListType string

const (
	BlockList IPAccessListType = "BLOCK"
	AllowList IPAccessListType = "ALLOW"
)

// Add an IP access list
type CreateIPAccessListRequest struct {
	Label       string           `json:"label,omitempty"`
	ListType    IPAccessListType `json:"list_type,omitempty"`
	IPAddresses []string         `json:"ip_addresses,omitempty"`
}

type IPAccessListStatus struct {
	ListID        string           `json:"list_id,omitempty"`
	Label         string           `json:"label,omitempty"`
	ListType      IPAccessListType `json:"list_type,omitempty"`
	IPAddresses   []string         `json:"ip_addresses,omitempty"`
	AddressCount  int              `json:"address_count,omitempty"`
	CreatedAt     int64            `json:"created_at,omitempty"`
	CreatorUserID int64            `json:"creator_user_id,omitempty"`
	UpdatedAt     int64            `json:"updated_at,omitempty"`
	UpdatorUserID int64            `json:"updator_user_id,omitempty"`
	Enabled       bool             `json:"enabled,omitempty"`
}

type IPAccessListStatusWrapper struct {
	IPAccessList IPAccessListStatus `json:"ip_access_list,omitempty"`
}

func ResourceIPAccessList() (_ *schema.Resource) {
	return &schema.Resource{
		Create: resourceIPACLCreate,
		Read:   resourceIPACLRead,
		Update: resourceIPACLUpdate,
		Delete: resourceIPACLDelete,

		Schema: map[string]*schema.Schema{
			"label": {
				Type:     schema.TypeString,
				Required: true,
			},
			"list_type": {
				Type:     schema.TypeString,
				Required: true,
				ValidateFunc: validation.StringInSlice(
					[]string{
						string(AllowList),
						string(BlockList),
					}, false),
			},
			"ip_addresses": {
				Type:     schema.TypeList,
				Required: true,
				Elem: &schema.Schema{
					Type:         schema.TypeString,
					ValidateFunc: validation.Any(validation.IsIPv4Address, validation.IsCIDR),
				},
			},
			"enabled": {
				Type:     schema.TypeBool,
				Optional: true,
				Default:  true,
			},
		},
	}
}

func resourceIPACLCreate(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*common.DatabricksClient)
	label := d.Get("label").(string)
	ipAddresses := d.Get("ip_addresses").([]interface{})
	listType := d.Get("list_type").(string)

	log.Println("IPACLLists: Calling IP ACL Create")
	status, err := NewIPAccessListsAPI(client).Create(
		internal.ConvertListInterfaceToString(ipAddresses),
		label,
		IPAccessListType(listType))
	log.Printf("IPACLLists: Created as  %v\n", status)
	if err != nil {
		log.Printf("IPACLLists:  Creation error %v\n", err)
		return
	}

	d.SetId(status.ListID)

	return resourceIPACLRead(d, m)
}

func resourceIPACLRead(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*common.DatabricksClient)
	status, err := NewIPAccessListsAPI(client).Read(d.Id())
	if err != nil {
		// check 404 (missing) and set id to empty string for tf
		if e, ok := err.(common.APIError); ok && e.IsMissing() {
			log.Printf("[IPACLLists:  missing resource due to error: %v\n", e)
			d.SetId("")
			return nil
		}
		return
	}

	return updateFromStatus(d, status)
}

func resourceIPACLUpdate(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*common.DatabricksClient)
	ipAddresses := internal.ConvertListInterfaceToString(d.Get("ip_addresses").([]interface{}))
	err = NewIPAccessListsAPI(client).Update(
		d.Id(),
		d.Get("label").(string),
		IPAccessListType(d.Get("list_type").(string)),
		ipAddresses,
		d.Get("enabled").(bool),
	)
	if err != nil {
		return
	}
	return resourceIPACLRead(d, m)
}

func resourceIPACLDelete(d *schema.ResourceData, m interface{}) (err error) {
	client := m.(*common.DatabricksClient)
	return NewIPAccessListsAPI(client).Delete(d.Id())
}

func updateFromStatus(d *schema.ResourceData, status IPAccessListStatus) (err error) {
	err = d.Set("label", status.Label)
	if err != nil {
		return
	}
	err = d.Set("list_type", string(status.ListType))
	if err != nil {
		return
	}
	err = d.Set("ip_addresses", status.IPAddresses)
	if err != nil {
		return
	}
	err = d.Set("enabled", status.Enabled)

	return
}
