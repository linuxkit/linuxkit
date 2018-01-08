package packngo

import "fmt"

const projectBasePath = "/projects"

// ProjectService interface defines available project methods
type ProjectService interface {
	List() ([]Project, *Response, error)
	Get(string) (*Project, *Response, error)
	Create(*ProjectCreateRequest) (*Project, *Response, error)
	Update(*ProjectUpdateRequest) (*Project, *Response, error)
	Delete(string) (*Response, error)
	ListVolumes(string) ([]Volume, *Response, error)
}

type volumesRoot struct {
	Volumes []Volume `json:"volumes"`
}

type projectsRoot struct {
	Projects []Project `json:"projects"`
}

// Project represents a Packet project
type Project struct {
	ID      string   `json:"id"`
	Name    string   `json:"name,omitempty"`
	Created string   `json:"created_at,omitempty"`
	Updated string   `json:"updated_at,omitempty"`
	Users   []User   `json:"members,omitempty"`
	Devices []Device `json:"devices,omitempty"`
	SSHKeys []SSHKey `json:"ssh_keys,omitempty"`
	URL     string   `json:"href,omitempty"`
}

func (p Project) String() string {
	return Stringify(p)
}

// ProjectCreateRequest type used to create a Packet project
type ProjectCreateRequest struct {
	Name          string `json:"name"`
	PaymentMethod string `json:"payment_method,omitempty"`
}

func (p ProjectCreateRequest) String() string {
	return Stringify(p)
}

// ProjectUpdateRequest type used to update a Packet project
type ProjectUpdateRequest struct {
	ID            string `json:"id"`
	Name          string `json:"name,omitempty"`
	PaymentMethod string `json:"payment_method,omitempty"`
}

func (p ProjectUpdateRequest) String() string {
	return Stringify(p)
}

// ProjectServiceOp implements ProjectService
type ProjectServiceOp struct {
	client *Client
}

// List returns the user's projects
func (s *ProjectServiceOp) List() ([]Project, *Response, error) {
	root := new(projectsRoot)

	resp, err := s.client.DoRequest("GET", projectBasePath, nil, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Projects, resp, err
}

// Get returns a project by id
func (s *ProjectServiceOp) Get(projectID string) (*Project, *Response, error) {
	path := fmt.Sprintf("%s/%s", projectBasePath, projectID)
	project := new(Project)

	resp, err := s.client.DoRequest("GET", path, nil, project)
	if err != nil {
		return nil, resp, err
	}

	return project, resp, err
}

// Create creates a new project
func (s *ProjectServiceOp) Create(createRequest *ProjectCreateRequest) (*Project, *Response, error) {
	project := new(Project)

	resp, err := s.client.DoRequest("POST", projectBasePath, createRequest, project)
	if err != nil {
		return nil, resp, err
	}

	return project, resp, err
}

// Update updates a project
func (s *ProjectServiceOp) Update(updateRequest *ProjectUpdateRequest) (*Project, *Response, error) {
	path := fmt.Sprintf("%s/%s", projectBasePath, updateRequest.ID)
	project := new(Project)

	resp, err := s.client.DoRequest("PATCH", path, updateRequest, project)
	if err != nil {
		return nil, resp, err
	}

	return project, resp, err
}

// Delete deletes a project
func (s *ProjectServiceOp) Delete(projectID string) (*Response, error) {
	path := fmt.Sprintf("%s/%s", projectBasePath, projectID)

	return s.client.DoRequest("DELETE", path, nil, nil)
}

// ListVolumes returns Volumes for a project
func (s *ProjectServiceOp) ListVolumes(projectID string) ([]Volume, *Response, error) {
	url := fmt.Sprintf("%s/%s%s", projectBasePath, projectID, volumeBasePath)
	root := new(volumesRoot)

	resp, err := s.client.DoRequest("GET", url, nil, root)
	if err != nil {
		return nil, resp, err
	}

	return root.Volumes, resp, err
}
