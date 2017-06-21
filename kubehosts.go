package main

import (
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"text/template"
)

type KubeHost struct {
	Hostname string
}

const shellHeader = `#!/bin/bash
#
# 404: Not found
#
# On mac os, you can run this command to add all these entries
#
#     bash <(curl -s {{Hostname}})
#
# Install hostess if you don't have it already
which hostess || brew install hostess


### These are domains we know. Hostess can add these to your hosts file
`

func renderScript(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	tmpl, err := template.New("kubehosts").Parse(shellHeader)
	if err != nil {
		panic(err)
	}
	err = tmpl.Execute(w, KubeHost{r.Host})
	if err != nil {
		panic(err)
	}

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	namespaceList, err := clientset.Namespaces().List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	namespaces := namespaceList.Items
	for n := range namespaces {
		ns := namespaces[n]
		if ns.Name != "kube-system" {
			fmt.Fprintf(w, "# Namespace: %s\n", ns.Name)

			ingressList, err := clientset.ExtensionsV1beta1Client.Ingresses(ns.Name).List(metav1.ListOptions{})
			if err != nil {
				panic(err.Error())
			}

			fmt.Printf("There are %d ingresses in the cluster\n", len(ingressList.Items))
			for i := range ingressList.Items {
				ingress := ingressList.Items[i]
				for r := range ingress.Spec.Rules {
					rule := ingress.Spec.Rules[r]
					fmt.Fprintf(w, "hostess add %s %s\n", rule.Host, ingress.Status.LoadBalancer.Ingress[0].IP)
				}
			}
			fmt.Fprint(w, "\n")
		}
	}

}

func renderHealth(w http.ResponseWriter, r *http.Request) {

	config, err := rest.InClusterConfig()
	if err != nil {
		panic(err.Error())
	}

	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	namespaceList, err := clientset.Namespaces().List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	namespaces := namespaceList.Items

	if len(namespaces) < 1 {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprintf(w, "No namespaces found")
	} else {
		w.WriteHeader(http.StatusOK)
		fmt.Fprintf(w, "All OK")
	}
}

func main() {
	http.HandleFunc("/", renderScript)
	http.HandleFunc("/healthz", renderHealth)
	http.ListenAndServe(":8080", nil)
}
