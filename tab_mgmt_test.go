package main

import (
	"sync"
	"testing"
)

func TestTabInsert(test *testing.T) {
	tabMan := newTabManager()
	defer tabMan.Shutdown()

	wg := &sync.WaitGroup{}
	wg.Add(5)
	var t0, t1, t2, t3, t4 *tabWithContext
	go func() {
		t0 = tabMan.Create(&tabContext{}, 0)
		wg.Done()
	}()
	go func() {
		t1 = tabMan.Create(&tabContext{}, 1)
		wg.Done()
	}()
	go func() {
		t2 = tabMan.Create(&tabContext{}, 2)
		wg.Done()
	}()
	go func() {
		t3 = tabMan.Create(&tabContext{}, 3)
		wg.Done()
	}()
	go func() {
		t4 = tabMan.Create(&tabContext{}, 4)
		wg.Done()
	}()
	wg.Wait()

	finder := func(index int) func(t *tabWithContext) bool {
		return func(t *tabWithContext) bool {
			if t.tab.Index() == index {
				return true
			}
			return false
		}
	}

	if t0 != tabMan.Find(finder(0)) {
		test.Fail()
	}
	if t1 != tabMan.Find(finder(1)) {
		test.Fail()
	}
	if t2 != tabMan.Find(finder(2)) {
		test.Fail()
	}
	if t3 != tabMan.Find(finder(3)) {
		test.Fail()
	}
	if t4 != tabMan.Find(finder(4)) {
		test.Fail()
	}
}

func TestTabDelete(test *testing.T) {
	tabMan := newTabManager()
	defer tabMan.Shutdown()

	wg := &sync.WaitGroup{}
	wg.Add(5)
	var t0, t1, t2, t3, t4 *tabWithContext
	go func() {
		t0 = tabMan.Create(&tabContext{}, 0)
		wg.Done()
	}()
	go func() {
		t1 = tabMan.Create(&tabContext{}, 1)
		wg.Done()
	}()
	go func() {
		t2 = tabMan.Create(&tabContext{}, 2)
		wg.Done()
	}()
	go func() {
		t3 = tabMan.Create(&tabContext{}, 3)
		wg.Done()
	}()
	go func() {
		t4 = tabMan.Create(&tabContext{}, 4)
		wg.Done()
	}()
	wg.Wait()

	tabMan.Delete(t2.tab)

	finder := func(index int) func(t *tabWithContext) bool {
		return func(t *tabWithContext) bool {
			if t.tab.Index() == index {
				return true
			}
			return false
		}
	}

	if t0 != tabMan.Find(finder(0)) {
		test.Fail()
	}
	if t1 != tabMan.Find(finder(1)) {
		test.Fail()
	}
	if nil != tabMan.Find(finder(2)) {
		test.Fail()
	}
	if t3 != tabMan.Find(finder(3)) {
		test.Fail()
	}
	if t4 != tabMan.Find(finder(4)) {
		test.Fail()
	}
}

func TestNoDuplicateTabInsert(test *testing.T) {
	tabMan := newTabManager()
	defer tabMan.Shutdown()

	wg := &sync.WaitGroup{}

	race := func() {
		tabMan.CreateIfNotFound(&tabContext{}, 0xff, func(t *tabWithContext) bool {
			return t.tab.Index() == 0xff
		})
		wg.Done()
	}

	horses := 100000
	wg.Add(horses)
	for i := 0; i < horses; i++ {
		go race()
	}
	wg.Wait()

	if len(tabMan.tabs) != 1 {
		printf(tabMan.tabs)
		test.Fail()
	}
}
