/*
Copyright (c) 2019 VMware, Inc. All Rights Reserved.

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

package object_test

import (
	"context"
	"fmt"
	"log"

	"github.com/vmware/govmomi/find"
	"github.com/vmware/govmomi/object"
	"github.com/vmware/govmomi/property"
	"github.com/vmware/govmomi/simulator"
	"github.com/vmware/govmomi/view"
	"github.com/vmware/govmomi/vim25"
	"github.com/vmware/govmomi/vim25/mo"
	"github.com/vmware/govmomi/vim25/types"
)

func ExampleVirtualMachine_HostSystem() {
	simulator.Example(func(ctx context.Context, c *vim25.Client) error {
		vm, err := find.NewFinder(c).VirtualMachine(ctx, "DC0_H0_VM0")
		if err != nil {
			return err
		}

		host, err := vm.HostSystem(ctx)
		if err != nil {
			return err
		}

		name, err := host.ObjectName(ctx)
		if err != nil {
			return err
		}

		fmt.Println(name)

		return nil
	})
	// Output: DC0_H0
}

func ExampleVirtualMachine_Clone() {
	simulator.Example(func(ctx context.Context, c *vim25.Client) error {
		finder := find.NewFinder(c)
		dc, err := finder.Datacenter(ctx, "DC0")
		if err != nil {
			return err
		}

		finder.SetDatacenter(dc)

		vm, err := finder.VirtualMachine(ctx, "DC0_H0_VM0")
		if err != nil {
			return err
		}

		folders, err := dc.Folders(ctx)
		if err != nil {
			return err
		}

		spec := types.VirtualMachineCloneSpec{
			PowerOn: false,
		}

		task, err := vm.Clone(ctx, folders.VmFolder, "example-clone", spec)
		if err != nil {
			return err
		}

		info, err := task.WaitForResult(ctx)
		if err != nil {
			return err
		}

		clone := object.NewVirtualMachine(c, info.Result.(types.ManagedObjectReference))
		name, err := clone.ObjectName(ctx)
		if err != nil {
			return err
		}

		fmt.Println(name)

		return nil
	})
	// Output: example-clone
}

func ExampleVirtualMachine_Reconfigure() {
	simulator.Example(func(ctx context.Context, c *vim25.Client) error {
		vm, err := find.NewFinder(c).VirtualMachine(ctx, "DC0_H0_VM0")
		if err != nil {
			return err
		}

		spec := types.VirtualMachineConfigSpec{Annotation: "example reconfig"}

		task, err := vm.Reconfigure(ctx, spec)
		if err != nil {
			return err
		}

		err = task.Wait(ctx)
		if err != nil {
			return err
		}

		var obj mo.VirtualMachine
		err = vm.Properties(ctx, vm.Reference(), []string{"config.annotation"}, &obj)
		if err != nil {
			return err
		}

		fmt.Println(obj.Config.Annotation)

		return nil
	})
	// Output: example reconfig
}

func ExampleCommon_Destroy() {
	model := simulator.VPX()
	model.Datastore = 2

	simulator.Example(func(ctx context.Context, c *vim25.Client) error {
		// Change to "LocalDS_0" will cause ResourceInUse error,
		// as simulator VMs created by the VPX model use "LocalDS_0".
		ds, err := find.NewFinder(c).Datastore(ctx, "LocalDS_1")
		if err != nil {
			return err
		}

		task, err := ds.Destroy(ctx)
		if err != nil {
			return err
		}

		if err = task.Wait(ctx); err != nil {
			return err
		}

		fmt.Println("destroyed", ds.InventoryPath)
		return nil
	}, model)
	// Output: destroyed /DC0/datastore/LocalDS_1
}

func ExampleCustomFieldsManager_Set() {
	simulator.Example(func(ctx context.Context, c *vim25.Client) error {
		m, err := object.GetCustomFieldsManager(c)
		if err != nil {
			return err
		}

		any := []string{"ManagedEntity"}
		field, err := m.Add(ctx, "backup", any[0], nil, nil) // adds the custom field "backup" to all types
		if err != nil {
			return err
		}

		v, err := view.NewManager(c).CreateContainerView(ctx, c.ServiceContent.RootFolder, any, true)
		if err != nil {
			log.Fatal(err)
		}

		all, err := v.Find(ctx, any, nil) // gives us the count of all objects in the inventory
		if err != nil {
			return err
		}

		refs, err := v.Find(ctx, []string{"VirtualMachine", "Datastore"}, nil)
		if err != nil {
			return err
		}

		for _, ref := range refs {
			err = m.Set(ctx, ref, field.Key, "true") // sets the custom value "backup=true" on specific types
			if err != nil {
				return err
			}
		}

		// filter used to find objects with "backup=true"
		filter := property.Filter{"customValue": &types.CustomFieldStringValue{
			CustomFieldValue: types.CustomFieldValue{Key: field.Key},
			Value:            "true",
		}}

		var objs []mo.ManagedEntity
		err = v.RetrieveWithFilter(ctx, any, []string{"name", "customValue"}, &objs, filter)
		if err != nil {
			return err
		}

		fmt.Printf("backup %d of %d objects", len(objs), len(all))
		return v.Destroy(ctx)
	})
	// Output: backup 5 of 22 objects
}
