package main

import (
	"flag"
	"fmt"
	"k8s.io/api/core/v1"
	"k8s.io/api/extensions/v1beta1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"k8s.io/client-go/tools/clientcmd"
	"k8s.io/client-go/util/homedir"
	"math/rand"
	"net/http"
	"path/filepath"
	"text/template"
	"time"
)

var kubeconfig *string

func configureKube() {
	if home := homedir.HomeDir(); home != "" {
		kubeconfig = flag.String("kubeconfig", filepath.Join(home, ".kube", "config"), "(optional) absolute path to the kubeconfig file")
	} else {
		kubeconfig = flag.String("kubeconfig", "", "absolute path to the kubeconfig file")
	}
	flag.Parse()
}

type KubeHost struct {
	Hostname string
}

const shellHeader = `#!/bin/bash
#
# 404: Not found
#
# On some platforms, you can run this command to add all these entries
#
#     bash <(curl -s {{.Hostname}})
#
# Supported platforms:
# - Mac OS i386 and x64
# - Linux i386 and x64
### Install hostess if you don't have it already
function installHostess() {
  ostype=unknown
  case "$(uname -s)" in
    Darwin) ostype=darwin ;;
    Linux)  ostype=linux ;;
  esac

  osarch=unknown
  case "$(uname -m)" in
    x86_64) osarch=amd64 ;;
    i386)   osarch=386 ;;
  esac

  if [ $ostype = unknown ]; then
    echo "Unknown OS. Install hostess manually. Look at https://github.com/cbednarski/hostess"
    exit 1
  fi
  if [ $osarch = unknown ]; then
    echo "Unknown Architecture. Install hostess manually. Look at https://github.com/cbednarski/hostess"
    exit 2
  fi

  mkdir -p ~/bin
  if [ ! -x ~/bin/hostess ]; then
    curl -L https://github.com/cbednarski/hostess/releases/download/v0.2.0/hostess_${ostype}_${osarch} > ~/bin/hostess
    chmod a+x ~/bin/hostess
  fi
  export PATH=$PATH:$HOME/bin
}

function h() {
  cat - \
    | sed -e "s/^Updated /$(tput setaf 4; tput dim)/g" \
    | sed -e "s/^Added /$(tput setaf 2; tput bold)/g" \
    | sed -e "s/ .*//g" \
    | sed -e "s/\./$(tput sgr0)./g"
}

function ns() {
  if [ $2 -gt 0 ]; then
    echo "$(tput setaf 3; tput bold)$1$(tput setaf 0; tput dim) - $2 ingresses$(tput sgr0)"
  else
    echo -n "$(tput setaf 0; tput dim)$1 - $2 ingresses$(tput sgr0)"
  fi
}
which hostess > /dev/null || installHostess
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
	config, err := getConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	namespaceList, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	namespaces := namespaceList.Items
	s := rand.NewSource(time.Now().Unix())
	rn := rand.New(s)
	for n := range namespaces {
		ns := namespaces[n]
		processNamespace(w, ns, clientset, rn)
	}
	fmt.Fprint(w, "\n")
}

func processNamespace(w http.ResponseWriter, ns v1.Namespace, clientset *kubernetes.Clientset, rn *rand.Rand) {
	ingressList, err := clientset.ExtensionsV1beta1().Ingresses(ns.Name).List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	fmt.Fprintf(w, "ns %s %d\n", ns.Name, len(ingressList.Items))
	if len(ingressList.Items) > 0 {
		fmt.Fprint(w, "f() {\n")
		for i := range ingressList.Items {
			ingress := ingressList.Items[i]
			processIngress(ingress, rn, w)
		}
		fmt.Fprint(w, "}\n")
		fmt.Fprint(w, "f | column\n")
	}
	fmt.Fprint(w, "echo ''")
	fmt.Fprint(w, "\n\n")
}

func processIngress(ingress v1beta1.Ingress, rn *rand.Rand, w http.ResponseWriter) {
	for r := range ingress.Spec.Rules {
		rule := ingress.Spec.Rules[r]
		ingresses := ingress.Status.LoadBalancer.Ingress
		index := rn.Intn(len(ingresses))
		balancerIngress := ingresses[index]
		fmt.Fprintf(w, "  hostess add %s %s | h\n", rule.Host, balancerIngress.IP)
	}
}

func renderHealth(w http.ResponseWriter, r *http.Request) {
	config, err := getConfig()
	if err != nil {
		panic(err.Error())
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		panic(err.Error())
	}
	namespaceList, err := clientset.CoreV1().Namespaces().List(metav1.ListOptions{})
	if err != nil {
		panic(err.Error())
	}
	namespaces := namespaceList.Items
	if len(namespaces) < 1 {
		w.WriteHeader(http.StatusInternalServerError)
		fmt.Fprint(w, "No namespaces found")
	} else {
		w.WriteHeader(http.StatusOK)
		fmt.Fprint(w, "All OK")
	}
}

func getConfig() (*rest.Config, error) {
	config, err := rest.InClusterConfig()
	if err == nil {
		return config, err
	}
	config, err = clientcmd.BuildConfigFromFlags("", *kubeconfig)
	if err != nil {
		panic(err.Error())
	}
	return config, err
}

func main() {
	configureKube()
	http.HandleFunc("/", renderScript)
	http.HandleFunc("/healthz", renderHealth)
	http.ListenAndServe(":8080", nil)
}
