/*
Copyright The KubeEdge Authors.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

// Code generated by client-gen. DO NOT EDIT.

package v1

import (
	"context"
	"time"

	v1 "github.com/kubeedge/sedna/pkg/apis/appawarecontroller/v1"
	scheme "github.com/kubeedge/sedna/pkg/client/clientset/versioned/scheme"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	types "k8s.io/apimachinery/pkg/types"
	watch "k8s.io/apimachinery/pkg/watch"
	rest "k8s.io/client-go/rest"
)

// AppawareHorizontalPodAutoscalersGetter has a method to return a AppawareHorizontalPodAutoscalerInterface.
// A group's client should implement this interface.
type AppawareHorizontalPodAutoscalersGetter interface {
	AppawareHorizontalPodAutoscalers(namespace string) AppawareHorizontalPodAutoscalerInterface
}

// AppawareHorizontalPodAutoscalerInterface has methods to work with AppawareHorizontalPodAutoscaler resources.
type AppawareHorizontalPodAutoscalerInterface interface {
	Create(ctx context.Context, appawareHorizontalPodAutoscaler *v1.AppawareHorizontalPodAutoscaler, opts metav1.CreateOptions) (*v1.AppawareHorizontalPodAutoscaler, error)
	Update(ctx context.Context, appawareHorizontalPodAutoscaler *v1.AppawareHorizontalPodAutoscaler, opts metav1.UpdateOptions) (*v1.AppawareHorizontalPodAutoscaler, error)
	UpdateStatus(ctx context.Context, appawareHorizontalPodAutoscaler *v1.AppawareHorizontalPodAutoscaler, opts metav1.UpdateOptions) (*v1.AppawareHorizontalPodAutoscaler, error)
	Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error
	DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error
	Get(ctx context.Context, name string, opts metav1.GetOptions) (*v1.AppawareHorizontalPodAutoscaler, error)
	List(ctx context.Context, opts metav1.ListOptions) (*v1.AppawareHorizontalPodAutoscalerList, error)
	Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error)
	Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.AppawareHorizontalPodAutoscaler, err error)
	AppawareHorizontalPodAutoscalerExpansion
}

// appawareHorizontalPodAutoscalers implements AppawareHorizontalPodAutoscalerInterface
type appawareHorizontalPodAutoscalers struct {
	client rest.Interface
	ns     string
}

// newAppawareHorizontalPodAutoscalers returns a AppawareHorizontalPodAutoscalers
func newAppawareHorizontalPodAutoscalers(c *AppawarecontrollerV1Client, namespace string) *appawareHorizontalPodAutoscalers {
	return &appawareHorizontalPodAutoscalers{
		client: c.RESTClient(),
		ns:     namespace,
	}
}

// Get takes name of the appawareHorizontalPodAutoscaler, and returns the corresponding appawareHorizontalPodAutoscaler object, and an error if there is any.
func (c *appawareHorizontalPodAutoscalers) Get(ctx context.Context, name string, options metav1.GetOptions) (result *v1.AppawareHorizontalPodAutoscaler, err error) {
	result = &v1.AppawareHorizontalPodAutoscaler{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("appawarehorizontalpodautoscalers").
		Name(name).
		VersionedParams(&options, scheme.ParameterCodec).
		Do(ctx).
		Into(result)
	return
}

// List takes label and field selectors, and returns the list of AppawareHorizontalPodAutoscalers that match those selectors.
func (c *appawareHorizontalPodAutoscalers) List(ctx context.Context, opts metav1.ListOptions) (result *v1.AppawareHorizontalPodAutoscalerList, err error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	result = &v1.AppawareHorizontalPodAutoscalerList{}
	err = c.client.Get().
		Namespace(c.ns).
		Resource("appawarehorizontalpodautoscalers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Do(ctx).
		Into(result)
	return
}

// Watch returns a watch.Interface that watches the requested appawareHorizontalPodAutoscalers.
func (c *appawareHorizontalPodAutoscalers) Watch(ctx context.Context, opts metav1.ListOptions) (watch.Interface, error) {
	var timeout time.Duration
	if opts.TimeoutSeconds != nil {
		timeout = time.Duration(*opts.TimeoutSeconds) * time.Second
	}
	opts.Watch = true
	return c.client.Get().
		Namespace(c.ns).
		Resource("appawarehorizontalpodautoscalers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Timeout(timeout).
		Watch(ctx)
}

// Create takes the representation of a appawareHorizontalPodAutoscaler and creates it.  Returns the server's representation of the appawareHorizontalPodAutoscaler, and an error, if there is any.
func (c *appawareHorizontalPodAutoscalers) Create(ctx context.Context, appawareHorizontalPodAutoscaler *v1.AppawareHorizontalPodAutoscaler, opts metav1.CreateOptions) (result *v1.AppawareHorizontalPodAutoscaler, err error) {
	result = &v1.AppawareHorizontalPodAutoscaler{}
	err = c.client.Post().
		Namespace(c.ns).
		Resource("appawarehorizontalpodautoscalers").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(appawareHorizontalPodAutoscaler).
		Do(ctx).
		Into(result)
	return
}

// Update takes the representation of a appawareHorizontalPodAutoscaler and updates it. Returns the server's representation of the appawareHorizontalPodAutoscaler, and an error, if there is any.
func (c *appawareHorizontalPodAutoscalers) Update(ctx context.Context, appawareHorizontalPodAutoscaler *v1.AppawareHorizontalPodAutoscaler, opts metav1.UpdateOptions) (result *v1.AppawareHorizontalPodAutoscaler, err error) {
	result = &v1.AppawareHorizontalPodAutoscaler{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("appawarehorizontalpodautoscalers").
		Name(appawareHorizontalPodAutoscaler.Name).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(appawareHorizontalPodAutoscaler).
		Do(ctx).
		Into(result)
	return
}

// UpdateStatus was generated because the type contains a Status member.
// Add a +genclient:noStatus comment above the type to avoid generating UpdateStatus().
func (c *appawareHorizontalPodAutoscalers) UpdateStatus(ctx context.Context, appawareHorizontalPodAutoscaler *v1.AppawareHorizontalPodAutoscaler, opts metav1.UpdateOptions) (result *v1.AppawareHorizontalPodAutoscaler, err error) {
	result = &v1.AppawareHorizontalPodAutoscaler{}
	err = c.client.Put().
		Namespace(c.ns).
		Resource("appawarehorizontalpodautoscalers").
		Name(appawareHorizontalPodAutoscaler.Name).
		SubResource("status").
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(appawareHorizontalPodAutoscaler).
		Do(ctx).
		Into(result)
	return
}

// Delete takes name of the appawareHorizontalPodAutoscaler and deletes it. Returns an error if one occurs.
func (c *appawareHorizontalPodAutoscalers) Delete(ctx context.Context, name string, opts metav1.DeleteOptions) error {
	return c.client.Delete().
		Namespace(c.ns).
		Resource("appawarehorizontalpodautoscalers").
		Name(name).
		Body(&opts).
		Do(ctx).
		Error()
}

// DeleteCollection deletes a collection of objects.
func (c *appawareHorizontalPodAutoscalers) DeleteCollection(ctx context.Context, opts metav1.DeleteOptions, listOpts metav1.ListOptions) error {
	var timeout time.Duration
	if listOpts.TimeoutSeconds != nil {
		timeout = time.Duration(*listOpts.TimeoutSeconds) * time.Second
	}
	return c.client.Delete().
		Namespace(c.ns).
		Resource("appawarehorizontalpodautoscalers").
		VersionedParams(&listOpts, scheme.ParameterCodec).
		Timeout(timeout).
		Body(&opts).
		Do(ctx).
		Error()
}

// Patch applies the patch and returns the patched appawareHorizontalPodAutoscaler.
func (c *appawareHorizontalPodAutoscalers) Patch(ctx context.Context, name string, pt types.PatchType, data []byte, opts metav1.PatchOptions, subresources ...string) (result *v1.AppawareHorizontalPodAutoscaler, err error) {
	result = &v1.AppawareHorizontalPodAutoscaler{}
	err = c.client.Patch(pt).
		Namespace(c.ns).
		Resource("appawarehorizontalpodautoscalers").
		Name(name).
		SubResource(subresources...).
		VersionedParams(&opts, scheme.ParameterCodec).
		Body(data).
		Do(ctx).
		Into(result)
	return
}
