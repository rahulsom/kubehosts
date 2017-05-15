package main

import (
	"fmt"
	"net/http"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

func handler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNotFound)
	fmt.Fprint(w, "#!/bin/bash\n")
	fmt.Fprint(w, "#\n")
	fmt.Fprint(w, "# 404: Not found\n")
	fmt.Fprint(w, "#\n")
	fmt.Fprint(w, "# On mac os, you can run this command to add all these entries\n")
	fmt.Fprint(w, "#\n")
	fmt.Fprintf(w, "#     bash <(curl -s %s)\n", r.Host)
	fmt.Fprint(w, "#\n")
	fmt.Fprint(w, "# Install hostess if you don't have it already\n")
	fmt.Fprint(w, "which hostess || brew install hostess\n")
	fmt.Fprint(w, "\n")
	fmt.Fprint(w, "### These are domains we know. Hostess can add these to your hosts file\n")

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

func main() {
	http.HandleFunc("/", handler)
	http.ListenAndServe(":8080", nil)
}
