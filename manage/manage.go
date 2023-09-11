package manage

import (
	"context"
	"crypto/x509"
	"fmt"
	"github.com/openziti/edge-api/rest_management_api_client"
	"github.com/openziti/edge-api/rest_management_api_client/identity"
	"github.com/openziti/edge-api/rest_management_api_client/service"
	"github.com/openziti/edge-api/rest_management_api_client/service_policy"
	"github.com/openziti/edge-api/rest_model"
	"github.com/openziti/edge-api/rest_util"
	"github.com/openziti/sdk-golang/ziti"
	"github.com/openziti/sdk-golang/ziti/enroll"
	"github.com/sirupsen/logrus"
	"os"
	"time"
)

var CtrlAddress string

var client *rest_management_api_client.ZitiEdgeManagement

func init() {
	zitiAdminUsername := os.Getenv("OPENZITI_USER")
	zitiAdminPassword := os.Getenv("OPENZITI_PWD")
	CtrlAddress = os.Getenv("OPENZITI_CTRL")

	if zitiAdminUsername == "" || zitiAdminPassword == "" || CtrlAddress == "" {
		if zitiAdminUsername == "" {
			logrus.Error("Please set the environment variable: OPENZITI_USER")
		}
		if zitiAdminPassword == "" {
			logrus.Error("Please set the environment variable: OPENZITI_PWD")
		}
		if CtrlAddress == "" {
			logrus.Error("Please set the environment variable: OPENZITI_CTRL")
		}
		logrus.Fatal("Cannot continue until these variables are set")
	}

	caCerts, err := rest_util.GetControllerWellKnownCas(CtrlAddress)
	if err != nil {
		logrus.Fatal(err)
	}
	caPool := x509.NewCertPool()
	for _, ca := range caCerts {
		caPool.AddCert(ca)
	}
	client, err = rest_util.NewEdgeManagementClientWithUpdb(zitiAdminUsername, zitiAdminPassword, CtrlAddress, caPool)
	if err != nil {
		logrus.Fatal(err)
	}
}

func FindIdentityDetail(identityID string) *rest_model.DetailIdentityEnvelope {
	// Retrieve and return the JWT token for the given identity ID
	params := &identity.DetailIdentityParams{
		Context: context.Background(),
		ID:      identityID,
	}
	params.SetTimeout(30 * time.Second)
	resp, err := client.Identity.DetailIdentity(params, nil)
	if err != nil {
		logrus.Fatal(err)
	}
	return resp.GetPayload()
}

func FindIdentity(identityName string) string {
	searchParam := identity.NewListIdentitiesParams()
	filter := "name = \"" + identityName + "\""
	searchParam.Filter = &filter
	id, err := client.Identity.ListIdentities(searchParam, nil)
	if err != nil {
		fmt.Println(err)
	}
	if id != nil && len(id.Payload.Data) == 0 {
		return ""
	}
	return *id.Payload.Data[0].ID
}

func DeleteIdentity(identityName string) {
	id := FindIdentity(identityName)
	if id == "" {
		return
	}
	// logic to delete reflect-server
	deleteParams := &identity.DeleteIdentityParams{
		ID: id,
	}
	deleteParams.SetTimeout(30 * time.Second)
	_, err := client.Identity.DeleteIdentity(deleteParams, nil)
	if err != nil {
		fmt.Println(err)
	}
}

func FindService(serviceName string) string {
	searchParam := service.NewListServicesParams()
	filter := "name=\"" + serviceName + "\""
	searchParam.Filter = &filter

	id, err := client.Service.ListServices(searchParam, nil)
	if err != nil {
		fmt.Println(err)
	}
	if id != nil && len(id.Payload.Data) == 0 {
		return ""
	}
	return *id.Payload.Data[0].ID
}

func DeleteService(serviceName string) {
	id := FindService(serviceName)
	if id == "" {
		return
	}

	deleteParams := &service.DeleteServiceParams{
		ID: id,
	}
	deleteParams.SetTimeout(30 * time.Second)
	_, err := client.Service.DeleteService(deleteParams, nil)
	if err != nil {
		fmt.Println(err)
	}
}

func FindServicePolicy(servicePolicyName string) string {
	searchParam := service_policy.NewListServicePoliciesParams()
	filter := "name=\"" + servicePolicyName + "\""
	searchParam.Filter = &filter

	id, err := client.ServicePolicy.ListServicePolicies(searchParam, nil)
	if err != nil {
		fmt.Println(err)
	}
	if id != nil && len(id.Payload.Data) == 0 {
		return ""
	}
	return *id.Payload.Data[0].ID
}

func DeleteServicePolicy(servicePolicyName string) {
	id := FindServicePolicy(servicePolicyName)
	if id == "" {
		return
	}

	deleteParams := &service_policy.DeleteServicePolicyParams{
		ID: id,
	}
	deleteParams.SetTimeout(30 * time.Second)
	_, err := client.ServicePolicy.DeleteServicePolicy(deleteParams, nil)
	if err != nil {
		fmt.Println(err)
	}
}

func CreateService(serviceName string, attribute string) rest_model.CreateLocation {
	encryptOn := true
	serviceCreate := &rest_model.ServiceCreate{
		//Configs:            serviceConfigs,
		EncryptionRequired: &encryptOn,
		Name:               &serviceName,
		RoleAttributes:     rest_model.Roles{attribute},
	}
	serviceParams := &service.CreateServiceParams{
		Service: serviceCreate,
		Context: context.Background(),
	}
	serviceParams.SetTimeout(30 * time.Second)
	resp, err := client.Service.CreateService(serviceParams, nil)
	if err != nil {
		fmt.Println(err)
		logrus.Fatal("Failed to create " + serviceName + " service")
	}
	return *resp.GetPayload().Data
}

func CreateIdentity(identType rest_model.IdentityType, identityName string, attributes string) *identity.CreateIdentityCreated {
	var isAdmin bool
	i := &rest_model.IdentityCreate{
		Enrollment: &rest_model.IdentityCreateEnrollment{
			Ott: true,
		},
		IsAdmin:                   &isAdmin,
		Name:                      &identityName,
		RoleAttributes:            &rest_model.Attributes{attributes},
		ServiceHostingCosts:       nil,
		ServiceHostingPrecedences: nil,
		Tags:                      nil,
		Type:                      &identType,
	}
	p := identity.NewCreateIdentityParams()
	p.Identity = i

	ident, err := client.Identity.CreateIdentity(p, nil)
	if err != nil {
		fmt.Println(err)
		logrus.Fatal("Failed to create the identity")
	}

	return ident
}

func EnrollIdentity(identityName string) *ziti.Config {
	identityID := FindIdentity(identityName)
	if identityID == "" {
		logrus.Fatal("identityID cant be found")
		return nil
	}

	params := &identity.DetailIdentityParams{
		Context: context.Background(),
		ID:      identityID,
	}
	params.SetTimeout(30 * time.Second)

	resp, err := client.Identity.DetailIdentity(params, nil)
	if err != nil {
		logrus.Fatal(err)
	}

	tkn, _, err := enroll.ParseToken(resp.GetPayload().Data.Enrollment.Ott.JWT)
	if err != nil {
		logrus.Fatal(err)
	}

	flags := enroll.EnrollmentFlags{
		Token:  tkn,
		KeyAlg: "RSA",
	}

	conf, err := enroll.Enroll(flags)
	if err != nil {
		logrus.Fatal(err)
	}

	return conf
}

func CreateServicePolicy(name string, servType rest_model.DialBind, identityRoles rest_model.Roles, serviceRoles rest_model.Roles) rest_model.CreateLocation {
	defaultSemantic := rest_model.SemanticAllOf
	servicePolicy := &rest_model.ServicePolicyCreate{
		IdentityRoles: identityRoles,
		Name:          &name,
		Semantic:      &defaultSemantic,
		ServiceRoles:  serviceRoles,
		Type:          &servType,
	}
	params := &service_policy.CreateServicePolicyParams{
		Policy:  servicePolicy,
		Context: context.Background(),
	}
	params.SetTimeout(30 * time.Second)
	resp, err := client.ServicePolicy.CreateServicePolicy(params, nil)
	if err != nil {
		fmt.Println(err)
		logrus.Fatal("Failed to create the " + name + " service policy")
	}

	return *resp.GetPayload().Data
}
