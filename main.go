/*
Copyright 2016 Juniper

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

*/
package main

import (
	"time"
	"fmt"
	"net"
	"io/ioutil"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
	"os"
	apiv1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
)


func main(){
	if len(os.Args) != 3 {
		panic("wrong number of args")
	}
	configMapName := os.Args[1]
	interfaceName := os.Args[2]
	prefix, err := externalIP(interfaceName)
	if err != nil {
		panic(err.Error())
	}
	err = createConfig(configMapName, prefix)
	if err != nil {
		panic(err.Error())
	}
}

func createConfig(configMapName string, prefix string) error{
	return retry(1, time.Second, func() error {
		nameSpaceByte, err := ioutil.ReadFile("/run/secrets/kubernetes.io/serviceaccount/namespace")
		if err != nil {
			panic(err.Error())
		}
		nameSpace := string(nameSpaceByte)
		config, err := rest.InClusterConfig()
		if err != nil {
			panic(err.Error())
		}
		clientset, err := kubernetes.NewForConfig(config)
		if err != nil {
			panic(err.Error())
		}
		configMap := &apiv1.ConfigMap{
			ObjectMeta: metav1.ObjectMeta{
				Name: configMapName,
				Namespace: nameSpace,
				},
			Data: map[string]string{"prefix":prefix},
		}
		configMapClient := clientset.CoreV1().ConfigMaps(nameSpace)
		cm, err := configMapClient.Get(configMapName, metav1.GetOptions{})
		if err != nil {
			configMapClient.Create(configMap)
			fmt.Println("created ", cm.Name)
		}
		fmt.Println("prefix: ", configMap.Data["prefix"])
		return nil
	})
}

func externalIP(interfaceName string) (string, error) {
	iface, err := net.InterfaceByName(interfaceName)
	if err != nil {
		return "", err
	}
	addresses, err := iface.Addrs()
	if err != nil {
		return "", err
	}
	for _, address := range(addresses){
		var ip net.IP
		switch v := address.(type) {
		case *net.IPNet:
			ip = v.IP
		case *net.IPAddr:
			ip = v.IP
		}
		if ip.To4() == nil{
			continue
		}
		return address.String(), nil
	}

	return "", errors.NewBadRequest("are you connected to the network?")
}
