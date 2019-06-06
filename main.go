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
)


func main(){
	if len(os.Args) != 2 {
		fmt.Println("number of args is not 1")
		panic("wrong number of args 1")
	}
	configMapName := os.Args[1]
	err := createConfig(configMapName)
	if err != nil {
		panic(err.Error())
	}
}

func createConfig(configMapName string) error{
	return retry(1, time.Second, func() error {
		ipMap, err := externalIP()
		if err != nil {
			panic(err.Error())
		}
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
			Data: ipMap,
		}
		configMapClient := clientset.CoreV1().ConfigMaps(nameSpace)
		fmt.Println(configMap)
		_, err = configMapClient.Get(configMapName, metav1.GetOptions{})
		if err != nil {
			configMapClient.Create(configMap)
			newCm, err := clientset.CoreV1().ConfigMaps(nameSpace).Create(configMap)
			if err != nil {
				panic(err.Error())
			}
			fmt.Println("created ", newCm.Name)
		}
		_, err = configMapClient.Get(configMapName, metav1.GetOptions{})
		if err != nil {
			fmt.Println("config map doesn't exist")
		} else {
			fmt.Println("config map exists")
		}
		fmt.Println("prefix: ", configMap.Data["prefix"])
		return nil
	})
}

func externalIP() (map[string]string, error) {
	var ipMap = make(map[string]string)
	ifaces, err := net.Interfaces()
	if err != nil {
		return nil, err
	}
	for _, iface := range ifaces {
		if iface.Flags&net.FlagUp == 0 {
			continue // interface down
		}
		if iface.Flags&net.FlagLoopback != 0 {
			continue // loopback interface
		}
		addrs, err := iface.Addrs()
		if err != nil {
			return nil, err
		}
		for _, addr := range addrs {
			var ip net.IP
			switch v := addr.(type) {
			case *net.IPNet:
				ip = v.IP
			case *net.IPAddr:
				ip = v.IP
			}
			if ip == nil || ip.IsLoopback() {
				continue
			}
			ip = ip.To4()
			if ip == nil {
				continue // not an ipv4 address
			}
			ipMap[iface.Name] = addr.String()
		}
	}
	return ipMap , nil
}
