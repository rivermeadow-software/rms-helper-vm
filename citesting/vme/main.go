package main

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"time"

	"github.com/gomorpheus/morpheus-go-sdk"
)

var (
	virtualImageName = "alpine-helper_kvm-v3.22-x86_64.iso"
)

func main() {

	config := VMEApplianceDeployConfig{
		VMEUrl:                "https://192.168.3.243",
		VerifySSL:             false,
		VMEUsername:           "rmadmin",
		VMEPassword:           "Password123#",
		VMEGroup:              "rmslab",
		VMECloud:              "vmecloud",
		VMECluster:            "vmecluster",
		VMENetwork:            "rmsmigration",
		VMEInternalNetwork:    "rmdemo",
		VMEDatastore:          "local",
		ApplianceInstanceName: "rmshelper",
		Environment:           "dev",
	}

	VMEDeploy(config)
}

func VMEDeploy(vmeConfig VMEApplianceDeployConfig) {
	var client *morpheus.Client
	if vmeConfig.VerifySSL {
		client = morpheus.NewClient(vmeConfig.VMEUrl)
	} else {
		client = morpheus.NewClient(vmeConfig.VMEUrl, morpheus.Insecure())
	}

	if vmeConfig.VMEToken != "" {
		client.SetAccessToken(vmeConfig.VMEToken, "", 86400, "write")
	} else {
		client.SetUsernameAndPassword(vmeConfig.VMEUsername, vmeConfig.VMEPassword)
	}
	resp, err := client.Login()
	if err != nil {
		fmt.Println("LOGIN ERROR: ", err)
	}
	fmt.Println("LOGIN RESPONSE:", resp)

	// find existing instance and delete it if it exists
	fmt.Println("Find and delete existing instance...")
	resp, err = client.ListInstances(&morpheus.Request{
		QueryParams: map[string]string{
			"name": vmeConfig.ApplianceInstanceName,
		},
	})
	if err != nil {
		log.Println("instance error:", err)
	}

	listResult := resp.Result.(*morpheus.ListInstancesResult)
	instanceCount := len(*listResult.Instances)
	if instanceCount == 1 {
		firstRecord := (*listResult.Instances)[0]
		instanceId := firstRecord.ID
		fmt.Println("Deleting existing instance with ID:", instanceId)
		client.DeleteInstance(instanceId, &morpheus.Request{
			QueryParams: map[string]string{
				"force": "on",
			},
		})
		// wait for 30 seconds to ensure deletion is complete
		fmt.Println("waiting 30 seconds to complete instance deletion")
		time.Sleep(30 * time.Second)
	}

	fmt.Println("Creating new virtual image...")
	// Update deployment stage
	//deployment.Stage = "upload"

	// find existing virtual image and delete it if it exists
	resp, err = client.ListVirtualImages(&morpheus.Request{
		QueryParams: map[string]string{
			"name": virtualImageName,
		},
	})
	if err != nil {
		log.Println("virtual image error:", err)
	}

	virtualImageListResult := resp.Result.(*morpheus.ListVirtualImagesResult)
	virtualImageCount := len(*virtualImageListResult.VirtualImages)
	if virtualImageCount == 1 {
		firstRecord := (*virtualImageListResult.VirtualImages)[0]
		virtualImageId := firstRecord.ID
		fmt.Println("Deleting existing virtual image with ID:", virtualImageId)
		client.DeleteVirtualImage(virtualImageId, &morpheus.Request{})
		// wait for 15 seconds to ensure deletion is complete
		fmt.Println("waiting 15 seconds to complete virtual image deletion")
		time.Sleep(15 * time.Second)
	}

	// // Read the entire file content into a byte slice.
	// content, err := os.ReadFile("config.yaml") // For Go versions older than 1.16, use ioutil.ReadFile.
	// if err != nil {
	// 	// Log the error and exit if the file cannot be read.
	// 	log.Fatal(err)
	// }

	// // Convert the byte slice to a string.
	// text := string(content)

	// cloudInitData := text

	// Create Virtual Image
	createReq := &morpheus.Request{
		Body: map[string]interface{}{
			"virtualImage": map[string]interface{}{
				"name":                 virtualImageName,
				"imageType":            "iso",
				"isCloudInit":          false,
				"installAgent":         false,
				"virtioSupported":      false,
				"isForceCustomization": false,
				"vmToolsInstalled":     false,
				//"osType":               "linux",
				//	"userData": cloudInitData,
			},
		},
	}

	createResp, err := client.CreateVirtualImage(createReq)

	if err != nil {
		fmt.Println(err)
	}

	createImageResult := createResp.Result.(*morpheus.CreateVirtualImageResult)
	fmt.Println(createImageResult)

	// Upload Virtual Image
	virtualImagePath := fmt.Sprintf("%s", virtualImageName)

	// Open the file to get an io.Reader
	data, err := os.Open(virtualImagePath)
	if err != nil {
		log.Fatalf("Error opening file: %v", err)
	}
	defer data.Close()

	uploadResp, err := client.UploadVirtualImage(createImageResult.VirtualImage.ID, &morpheus.Request{
		QueryParams: map[string]string{
			"filename": virtualImageName,
		},
		IsStream:   true,
		StreamBody: data,
	})

	if err != nil {
		log.Println(err)
	}

	uploadImageResult := uploadResp.Result.(*morpheus.UploadVirtualImageResult)
	fmt.Println(uploadImageResult)

	fmt.Println("Creating new instance from virtual image...")

	// Create Instance
	// Config
	config := make(map[string]interface{})

	// Resource Pool
	resourcePoolResp, err := client.Execute(&morpheus.Request{
		Method:      "GET",
		Path:        "/api/options/zonePools",
		QueryParams: map[string]string{},
	})
	if err != nil {
		log.Println(err)
	}

	var itemResponsePayload ResourcePoolOptions
	json.Unmarshal(resourcePoolResp.Body, &itemResponsePayload)
	var resourcePoolId int
	for _, v := range itemResponsePayload.Data {
		if v.ProviderType == "mvm" && v.Name == vmeConfig.VMECluster {
			resourcePoolId = v.Id
		}
	}

	config["resourcePoolId"] = resourcePoolId
	config["poolProviderType"] = "mvm"

	// Image ID
	config["imageId"] = createImageResult.VirtualImage.ID

	// Host Id
	//config["kvmHostId"] = s.builder.config.HostID

	// Attach VirtIO Drivers
	config["attachVirtIODrivers"] = false

	// Skip Agent Install
	config["noAgent"] = true

	// Skip Backup Creation
	config["createBackup"] = false

	groupResp, err := client.FindGroupByName(vmeConfig.VMEGroup)
	if err != nil {
		log.Printf("API FAILURE: %s - %s", groupResp, err)
	}
	group := groupResp.Result.(*morpheus.GetGroupResult)

	instancePayload := map[string]interface{}{
		"name":            vmeConfig.ApplianceInstanceName,
		"type":            "mvm",
		"description":     "RMS Helper",
		"instanceContext": vmeConfig.Environment,
		"instanceType": map[string]interface{}{
			"code": "mvm",
		},
		"site": map[string]interface{}{
			"id": group.Group.ID,
		},
		"plan": map[string]interface{}{
			"id": 19,
		},
		// How to find the instance layout id
		"layout": map[string]interface{}{
			//	"name":              "Single HPE VM",
			//	"provisionTypeCode": "kvm",
			"id": 32,
		},
	}

	cloudResp, err := client.FindCloudByName(vmeConfig.VMECloud)
	if err != nil {
		log.Printf("API FAILURE: %s - %s", cloudResp, err)
	}
	cloud := cloudResp.Result.(*morpheus.GetCloudResult)

	payload := map[string]interface{}{
		"zoneId":   cloud.Cloud.ID,
		"instance": instancePayload,
		"config":   config,
	}

	// Instance Network Configuration
	var Nics []PayloadNetworkInterface
	resp, err = client.ListNetworks(&morpheus.Request{
		QueryParams: map[string]string{
			"name": vmeConfig.VMENetwork,
		},
	})
	if err != nil {
		log.Printf("API FAILURE: %s - %s", cloudResp, err)
	}
	networks := resp.Result.(*morpheus.ListNetworksResult)
	networkId := 0
	for _, network := range *networks.Networks {
		if network.ZonePool.Name == vmeConfig.VMECluster {
			networkId = int(network.ID)
		}
	}

	// Error out if the defined network is unable to be found
	if networkId == 0 {
		log.Printf("Unable to find network named %s", vmeConfig.VMENetwork)
	}
	var NetworkData PayloadNetworkInterface
	//NetworkData.NetworkInterfaceTypeID = nic.NetworkInterfaceTypeId
	NetworkData.IPMode = "dhcp"
	//NetworkData.IPAddress = "192.168.3.245"
	NetworkData.Network.ID = fmt.Sprintf("network-%d", networkId)
	Nics = append(Nics, NetworkData)

	// Add Nics and Volumes to payload
	payload["networkInterfaces"] = Nics

	// Storage Volumes
	var Volumes []PayloadStorageVolume

	var StorageDemo PayloadStorageVolume
	StorageDemo.ID = -1
	StorageDemo.Name = "root"
	StorageDemo.RootVolume = true

	// Define size of volume in GB
	StorageDemo.Size = int64(10) // 10 GB
	//StorageDemo.StorageType = sv.StorageTypeID
	StorageDemo.DatastoreId = "1"
	Volumes = append(Volumes, StorageDemo)

	payload["volumes"] = Volumes
	payload["layoutSize"] = 1

	req := &morpheus.Request{Body: payload}
	createInstanceresp, err := client.CreateInstance(req)
	if err != nil {
		log.Printf("API FAILURE: %s - %s", createInstanceresp, err)
	}
	log.Printf("API RESPONSE: %s", createInstanceresp)
	result := createInstanceresp.Result.(*morpheus.CreateInstanceResult)
	instance := result.Instance
	fmt.Println("Created instance: ", instance)

	// Status List: provisioning, pending, removing
	// Poll Instance for status
	currentStatus := "provisioning"
	completedStatuses := []string{"running", "failed", "warning", "denied", "cancelled", "suspended"}
	fmt.Printf("Waiting for instance (%d) to become ready", instance.ID)

	for !stringInSlice(completedStatuses, currentStatus) {
		// sleep 5 seconds between polls
		time.Sleep(5 * time.Second)
		resp, err := client.GetInstance(instance.ID, &morpheus.Request{})
		if err != nil {
			log.Println("API ERROR: ", err)
		}
		result := resp.Result.(*morpheus.GetInstanceResult)
		currentStatus = result.Instance.Status
		fmt.Printf("Waiting for instance to provision - %s\n", currentStatus)
	}

	fmt.Printf("Instance (%d) has become ready", instance.ID)
	respGet, err := client.GetInstance(instance.ID, req)
	if err != nil {
		fmt.Println(err.Error())
		log.Printf("API FAILURE: %s - %s", respGet, err)
	}
	log.Printf("API RESPONSE: %s", respGet)
	resultGet := respGet.Result.(*morpheus.GetInstanceResult)
	instanceGet := resultGet.Instance
	fmt.Println(instanceGet)
}

type VMEOutDetails struct {
	Groups     []string `json:"groups"`
	Clouds     []string `json:"clouds"`
	Networks   []string `json:"networks"`
	Clusters   []string `json:"clusters"`
	Datastores []string `json:"datastores"`
}

type ResourcePoolOptions struct {
	Success bool `json:"success"`
	Data    []struct {
		Id           int    `json:"id"`
		Name         string `json:"name"`
		IsGroup      bool   `json:"isGroup"`
		Group        string `json:"group"`
		IsDefault    bool   `json:"isDefault"`
		Type         string `json:"type"`
		ProviderType string `json:"providerType"`
		Value        string `json:"value"`
	} `json:"data"`
}

func stringInSlice(s []string, str string) bool {
	for _, v := range s {
		if v == str {
			return true
		}
	}

	return false
}

type PayloadNetworkInterface struct {
	Network struct {
		ID string `json:"id"`
	} `json:"network"`
	IPMode                 string `json:"ipMode"`
	IPAddress              string `json:"ipAddress"`
	NetworkInterfaceTypeID int64  `json:"networkInterfaceTypeID"`
}

type PayloadStorageVolume struct {
	ID          int64  `json:"id"`
	RootVolume  bool   `json:"rootVolume"`
	Name        string `json:"name"`
	Size        int64  `json:"size"`
	StorageType int64  `json:"storageType"`
	DatastoreId string `json:"datastoreId"`
}

type VMEApplianceDeployConfig struct {
	VMEUrl                string   `json:"url"`
	VerifySSL             bool     `json:"verify_ssl"`
	VMEUsername           string   `json:"username"`
	VMEPassword           string   `json:"password"`
	VMEToken              string   `json:"token"`
	VMEGroup              string   `json:"group"`
	VMECloud              string   `json:"cloud"`
	VMECluster            string   `json:"cluster"`
	VMENetwork            string   `json:"network"`
	VMEInternalNetwork    string   `json:"internal_network"`
	VMEDatastore          string   `json:"datastore"`
	ApplianceInstanceName string   `json:"instanceName"`
	Environment           string   `json:"environment"`
	InstancePlan          string   `json:"instance_plan"`
	IPAddressType         string   `json:"ip_address_type"`
	IPAddress             string   `json:"ip_address"`
	SubnetMask            string   `json:"subnet_mask"`
	DefaultGateway        string   `json:"default_gateway"`
	DNSServers            []string `json:"dns_servers"`
	ProxyURL              string   `json:"proxy_url"`
	ProxyPort             string   `json:"proxy_port"`
	ProxyUsername         string   `json:"proxy_username"`
	ProxyPassword         string   `json:"proxy_password"`
}
