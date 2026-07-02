package bitbucket

// Commit represents a commit associated with a pull request.
type Commit struct {
	Hash    string `json:"hash"`
	Message string `json:"message"`
	Date    string `json:"date"`
	Author  struct {
		Raw string `json:"raw"`
	} `json:"author"`
}

// CommitsResponse is the API response for pull request commits.
type CommitsResponse struct {
	Values []Commit `json:"values"`
	Next   string   `json:"next"`
}

// CommitStatus represents a build/deployment/status associated with a commit.
type CommitStatus struct {
	State       string `json:"state"`
	Key         string `json:"key"`
	Name        string `json:"name"`
	URL         string `json:"url"`
	Description string `json:"description"`
	CreatedOn   string `json:"created_on"`
	UpdatedOn   string `json:"updated_on"`
	Type        string `json:"type"`
}

// CommitStatusesResponse is the API response for commit statuses.
type CommitStatusesResponse struct {
	Values []CommitStatus `json:"values"`
	Next   string         `json:"next"`
}

// Comment represents a pull request comment.
type Comment struct {
	ID      int `json:"id"`
	Content struct {
		Raw string `json:"raw"`
	} `json:"content"`
	User struct {
		DisplayName string `json:"display_name"`
		UUID        string `json:"uuid"`
	} `json:"user"`
	CreatedOn string `json:"created_on"`
	UpdatedOn string `json:"updated_on"`
	Inline    *struct {
		Path string `json:"path"`
		From int    `json:"from,omitempty"`
		To   int    `json:"to,omitempty"`
	} `json:"inline,omitempty"`
	Parent *struct {
		ID int `json:"id"`
	} `json:"parent,omitempty"`
	Resolved bool `json:"resolved,omitempty"`
}

// CommentsResponse is the API response for pull request comments.
type CommentsResponse struct {
	Values []Comment `json:"values"`
	Next   string    `json:"next"`
}

// Pipeline represents a Bitbucket Pipeline run.
type Pipeline struct {
	UUID        string `json:"uuid"`
	BuildNumber int    `json:"build_number"`
	CreatedOn   string `json:"created_on"`
	CompletedOn string `json:"completed_on"`
	State       struct {
		Name  string `json:"name"`
		Stage *struct {
			Name string `json:"name"`
		} `json:"stage"`
		Result *struct {
			Name string `json:"name"`
		} `json:"result"`
	} `json:"state"`
	Target struct {
		RefName string `json:"ref_name"`
		RefType string `json:"ref_type"`
		Commit  *struct {
			Hash string `json:"hash"`
		} `json:"commit"`
	} `json:"target"`
	Trigger struct {
		Name string `json:"name"`
	} `json:"trigger"`
	Creator *struct {
		DisplayName string `json:"display_name"`
	} `json:"creator"`
	BuildSecondsUsed int `json:"build_seconds_used"`
	Links            struct {
		Self struct {
			Href string `json:"href"`
		} `json:"self"`
	} `json:"links"`
}

// PipelinesResponse is the paged response for listing pipelines.
type PipelinesResponse struct {
	Values []Pipeline `json:"values"`
	Next   string     `json:"next"`
}

// PipelineStep represents a single step in a pipeline.
type PipelineStep struct {
	UUID              string `json:"uuid"`
	Name              string `json:"name"`
	RunNumber         int    `json:"run_number"`
	CreatedOn         string `json:"created_on"`
	CompletedOn       string `json:"completed_on"`
	DurationInSeconds int    `json:"duration_in_seconds"`
	State             struct {
		Name   string `json:"name"`
		Result *struct {
			Name string `json:"name"`
		} `json:"result"`
	} `json:"state"`
	Image *struct {
		Name string `json:"name"`
	} `json:"image"`
	SetupCommands  []PipelineCommand `json:"setup_commands"`
	ScriptCommands []PipelineCommand `json:"script_commands"`
}

// PipelineCommand represents a command within a step.
type PipelineCommand struct {
	Name    string `json:"name"`
	Action  string `json:"action"`
	Command string `json:"command"`
}

// PipelineStepsResponse is the paged response for listing pipeline steps.
type PipelineStepsResponse struct {
	Values []PipelineStep `json:"values"`
	Next   string         `json:"next"`
}

