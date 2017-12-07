package main // import "github.com/mojlighetsministeriet/swarm-info"
import (
	"context"
	"net/http"
	"strings"
	"time"

	"github.com/docker/docker/api/types"
	"github.com/docker/docker/client"
	"github.com/labstack/echo"
	"github.com/labstack/echo/middleware"
	"github.com/labstack/gommon/log"
	"github.com/mojlighetsministeriet/utils"
)

// Container represents a docker container running on a node
type Container struct {
	ID        string    `json:"id"`
	Image     string    `json:"image"`
	ImageHash string    `json:"imageHash"`
	Error     string    `json:"error,omitempty"`
	State     string    `json:"state"`
	ServiceID string    `json:"serviceId"`
	NodeID    string    `json:"nodeId"`
	CreatedAt time.Time `json:"createdAt"`
}

// Node represents a docker service running on a machine
type Node struct {
	ID         string      `json:"id"`
	Hostname   string      `json:"hostname"`
	State      string      `json:"state"`
	Manager    bool        `json:"manager"`
	IP         string      `json:"ip"`
	Containers []Container `json:"containers,omitempty"`
	JoinedAt   time.Time   `json:"joinedAt"`
}

// Service represents a docker swarm service
type Service struct {
	ID         string      `json:"id"`
	Name       string      `json:"name"`
	Replicas   *uint64     `json:"replicas"`
	Containers []Container `json:"containers,omitempty"`
	CreatedAt  time.Time   `json:"createdAt"`
}

// Swarm represents a cluster of docker nodes
type Swarm struct {
	Nodes      []Node      `json:"nodes,omitempty"`
	Services   []Service   `json:"services,omitempty"`
	Containers []Container `json:"containers,omitempty"`
}

// GetNodeByID returns a pointer to a node given it's id
func (swarm *Swarm) GetNodeByID(id string) (node *Node) {
	for index := 0; index < len(swarm.Nodes); index++ {
		if swarm.Nodes[index].ID == id {
			node = &swarm.Nodes[index]
			return
		}
	}

	return
}

// GetServiceByID returns a pointer to a service given it's id
func (swarm *Swarm) GetServiceByID(id string) (service *Service) {
	for index := 0; index < len(swarm.Services); index++ {
		if swarm.Services[index].ID == id {
			service = &swarm.Services[index]
			return
		}
	}

	return
}

// GetContainerByID returns a pointer to a container given it's id
func (swarm *Swarm) GetContainerByID(id string) (container *Container) {
	for index := 0; index < len(swarm.Containers); index++ {
		if swarm.Containers[index].ID == id {
			container = &swarm.Containers[index]
			return
		}
	}

	return
}

var swarm Swarm

func updateSwarm(cli *client.Client, logger echo.Logger) {
	nodeList, err := cli.NodeList(context.Background(), types.NodeListOptions{})
	if err != nil {
		logger.Error(err)
	}

	newNodes := []Node{}
	for _, nodeInfo := range nodeList {
		node := Node{
			ID:       nodeInfo.ID,
			Hostname: nodeInfo.Description.Hostname,
			State:    string(nodeInfo.Status.State),
			Manager:  nodeInfo.ManagerStatus.Leader,
			IP:       nodeInfo.Status.Addr,
			JoinedAt: nodeInfo.CreatedAt,
		}
		newNodes = append(newNodes, node)
	}

	taskList, err := cli.TaskList(context.Background(), types.TaskListOptions{})
	if err != nil {
		logger.Error(err)
	}

	newContainers := []Container{}
	for _, task := range taskList {
		imageParts := strings.Split(task.Spec.ContainerSpec.Image, "@")
		container := Container{
			ID:        task.Status.ContainerStatus.ContainerID,
			Image:     imageParts[0],
			ImageHash: imageParts[1],
			Error:     task.Status.Err,
			State:     string(task.Status.State),
			ServiceID: task.ServiceID,
			NodeID:    task.NodeID,
			CreatedAt: task.CreatedAt,
		}
		newContainers = append(newContainers, container)
	}

	serviceList, err := cli.ServiceList(context.Background(), types.ServiceListOptions{})
	if err != nil {
		logger.Error(err)
	}

	newServices := []Service{}
	for _, serviceInfo := range serviceList {
		service := Service{
			ID:        serviceInfo.ID,
			Name:      serviceInfo.Spec.Name,
			Replicas:  serviceInfo.Spec.Mode.Replicated.Replicas,
			CreatedAt: serviceInfo.CreatedAt,
		}

		newServices = append(newServices, service)
	}

	swarm.Nodes = newNodes
	swarm.Containers = newContainers
	swarm.Services = newServices
}

func main() {
	swarm = Swarm{}

	cli, err := client.NewEnvClient()
	if err != nil {
		panic(err)
	}

	service := echo.New()
	service.Use(middleware.AddTrailingSlashWithConfig(middleware.TrailingSlashConfig{
		RedirectCode: http.StatusMovedPermanently,
	}))
	service.Use(middleware.Gzip())
	service.Logger.SetLevel(log.INFO)

	go func() {
		updateSwarm(cli, service.Logger)
		time.Sleep(5 * time.Second)
	}()

	service.GET("/api/node/", func(request echo.Context) error {
		return request.JSON(200, swarm.Nodes)
	})

	service.GET("/api/container/", func(request echo.Context) error {
		return request.JSON(200, swarm.Containers)
	})

	service.GET("/api/container/:id/logs/", func(request echo.Context) error {
		logs, err := cli.ContainerLogs(context.Background(), request.Param("id"), types.ContainerLogsOptions{ShowStdout: true})
		if err != nil {
			return request.String(404, "Not Found")
		}
		return request.Stream(200, "text/plain;charset=UTF-8", logs)
	})

	service.GET("/api/service/", func(request echo.Context) error {
		return request.JSON(200, swarm.Services)
	})

	service.GET("/api/service/:id/logs/", func(request echo.Context) error {
		logs, err := cli.ServiceLogs(context.Background(), request.Param("id"), types.ContainerLogsOptions{})
		if err != nil {
			return request.String(404, "Not Found")
		}
		return request.Stream(200, "text/plain;charset=UTF-8", logs)
	})

	service.GET("/api/aggrigate/", func(request echo.Context) error {
		swarmTree := Swarm{}
		swarmTree.Containers = swarm.Containers
		swarmTree.Nodes = swarm.Nodes
		swarmTree.Services = swarm.Services

		for _, container := range swarmTree.Containers {
			node := swarmTree.GetNodeByID(container.NodeID)
			if node != nil {
				node.Containers = append(node.Containers, container)
			}

			service := swarmTree.GetServiceByID(container.ServiceID)
			if service != nil {
				service.Containers = append(service.Containers, container)
			}
		}

		swarmTree.Containers = nil
		return request.JSON(200, swarmTree)
	})

	type routeInfo struct {
		Path   string `json:"path"`
		Method string `json:"method"`
	}
	var registeredRoutes []routeInfo
	for _, route := range service.Routes() {
		if !strings.HasSuffix(route.Path, "/*") {
			registeredRoute := routeInfo{
				Path:   route.Path,
				Method: route.Method,
			}
			registeredRoutes = append(registeredRoutes, registeredRoute)
		}
	}

	service.GET("/api/", func(context echo.Context) error {
		return context.JSON(http.StatusOK, registeredRoutes)
	})

	service.Logger.Fatal(service.Start(":" + utils.GetEnv("PORT", "80")))
}