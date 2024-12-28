package rest

type ResourceMetadata struct {
	Annotations       map[string]string `json:"annotations" yaml:"annotations"`
	Labels            map[string]string `json:"labels" yaml:"labels"`
	Name              string            `json:"name" yaml:"name"`
	Namespace         string            `json:"namespace" yaml:"namespace"`
	CreationTimestamp string            `json:"creationTimestamp" yaml:"creationTimestamp"`
	UID               string            `json:"uid" yaml:"uid"`
}

type TargetRef struct {
	Kind      string `json:"kind" yaml:"kind"`
	Name      string `json:"name" yaml:"name"`
	Namespace string `json:"namespace" yaml:"namespace"`
	UID       string `json:"uid" yaml:"uid"`
}
