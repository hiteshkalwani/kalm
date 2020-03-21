package handler

import (
	"github.com/kapp-staging/kapp/api/client"
	"github.com/labstack/echo/v4"
	"github.com/sirupsen/logrus"
	v1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/fields"
	"k8s.io/apimachinery/pkg/labels"
)

var (
	ListAll = v1.ListOptions{
		LabelSelector: labels.Everything().String(),
		FieldSelector: fields.Everything().String(),
	}
)

type ApiHandler struct {
	clientManager *client.ClientManager
	logger        *logrus.Logger
}

type H map[string]interface{}

func (h *ApiHandler) Install(e *echo.Echo) {
	e.GET("/ping", handlePing)

	// permission routes
	e.POST("/login", h.handleLogin)
	e.GET("/login/status", h.handleLoginStatus)

	// original resources routes
	gV1 := e.Group("/v1", h.AuthClientMiddleware)
	gV1.GET("/deployments", h.handleGetDeployments)
	gV1.GET("/nodes", h.handleGetNodes)
	gV1.GET("/nodes/metrics", h.handleGetNodeMetrics)
	gV1.GET("/persistentvolumes", h.handleGetPVs)

	// kapp resources
	gV1.GET("/componenttemplates", h.handleGetComponentTemplates)
	gV1.POST("/componenttemplates", h.handleCreateComponentTemplate)
	gV1.PUT("/componenttemplates/:name", h.handleUpdateComponentTemplate)
	gV1.DELETE("/componenttemplates/:name", h.handleDeleteComponentTemplate)

	gV1.GET("/dependencies", h.handleGetDependencies)
	gV1.GET("/dependencies/available", h.handleGetAvailableDependencies)

	gV1.GET("/applications", h.handleGetApplicationsOld)
	gV1.GET("/applications/:namespace", h.handleGetApplicationsOld)
	gV1.POST("/applications/:namespace", h.handleCreateApplication)
	gV1.PUT("/applications/:namespace/:name", h.handleUpdateApplication)
	gV1.DELETE("/applications/:namespace/:name", h.handleDeleteApplication)

	gV1.GET("/files", h.handleGetFiles)
	gV1.POST("/files", h.handleCreateFile)
	gV1.PUT("/files/:name", h.handleUpdateFile)
	gV1.DELETE("/files/:name", h.handleDeleteFile)

	gV1.GET("/clusterroles", h.handleGetClusterRoles)
	gV1.POST("/clusterroles", h.handleCreateClusterRoles)

	gV1.GET("/clusterrolebindings", h.handleGetClusterRoleBindings)
	gV1.POST("/clusterrolebindings", h.handleCreateClusterRoleBindings)

	gV1.GET("/serviceaccounts", h.handleGetServiceAccounts)
	gV1.POST("/serviceaccounts", h.handleCreateServiceAccounts)

	gV1.GET("/secrets", h.handleGetSecrets)
	gV1.POST("/secrets", h.handleCreateSecrets)

	gV1Alpha1 := e.Group("/v1alpha1", h.AuthClientMiddleware)
	gV1Alpha1.GET("/applications", h.handleGetApplications)
	gV1Alpha1.GET("/applications/:namespace", h.handleGetApplications)
	gV1Alpha1.GET("/applications/:namespace/:name", h.handleGetApplicationDetails)
	gV1Alpha1.PUT("/applications/:namespace/:name", h.handleUpdateApplicationNew)
	gV1Alpha1.DELETE("/applications/:namespace/:name", h.handleDeleteApplication)
	gV1Alpha1.POST("/applications/:namespace", h.handleCreateApplicationNew)

	gV1Alpha1.GET("/componenttemplates", h.handleGetComponentTemplatesNew)
	gV1Alpha1.POST("/componenttemplates", h.handleCreateComponentTemplateNew)
	gV1Alpha1.PUT("/componenttemplates/:name", h.handleUpdateComponentTemplateNew)
	gV1Alpha1.DELETE("/componenttemplates/:name", h.handleDeleteComponentTemplateNew)
}

func NewApiHandler(clientManager *client.ClientManager) *ApiHandler {
	return &ApiHandler{
		clientManager: clientManager,
		logger:        logrus.New(),
	}
}