// Copyright 2015-2016 trivago GmbH
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package shared

import (
	"sync"
	"testing"
)

func getMockMetric() metrics {
	return metrics{
		new(sync.RWMutex),
		make(map[string]*int64),
	}
}

func TestMetricsSet(t *testing.T) {
	expect := NewExpect(t)
	mockMetric := getMockMetric()

	// test for initialization to zero
	mockMetric.New("MockMetric")
	count, err := mockMetric.Get("MockMetric")
	expect.Nil(err)
	expect.Equal(int64(0), count)

	// test for setting to a particular value
	mockMetric.Set("MockMetric", int64(5))
	count, err = mockMetric.Get("MockMetric")
	expect.Nil(err)
	expect.Equal(int64(5), count)

	// test for setting to a particular int
	mockMetric.SetI("MockMetric", 5)
	count, err = mockMetric.Get("MockMetric")
	expect.Nil(err)
	expect.Equal(int64(5), count)

	// test for setting to a particular float
	mockMetric.SetF("MockMetric", 4.3)
	count, err = mockMetric.Get("MockMetric")
	expect.Nil(err)
	expect.Equal(int64(4), count)
}

func TestMetricsAddSub(t *testing.T) {
	expect := NewExpect(t)
	mockMetric := getMockMetric()

	mockMetric.New("MockMetric")
	mockMetric.Add("MockMetric", int64(1))
	count, err := mockMetric.Get("MockMetric")
	expect.Nil(err)
	expect.Equal(int64(1), count)

	mockMetric.AddI("MockMetric", 1)
	count, err = mockMetric.Get("MockMetric")
	expect.Nil(err)
	expect.Equal(int64(2), count)

	mockMetric.AddF("MockMetric", 2.4)
	count, err = mockMetric.Get("MockMetric")
	expect.Nil(err)
	expect.Equal(int64(4), count)

	mockMetric.Sub("MockMetric", int64(1))
	count, err = mockMetric.Get("MockMetric")
	expect.Nil(err)
	expect.Equal(int64(3), count)

	mockMetric.SubF("MockMetric", 1.6)
	count, err = mockMetric.Get("MockMetric")
	expect.Nil(err)
	expect.Equal(int64(1), count)

	mockMetric.SubI("MockMetric", 1)
	count, err = mockMetric.Get("MockMetric")
	expect.Nil(err)
	expect.Equal(int64(0), count)
}

func TestMetricsIncDec(t *testing.T) {
	expect := NewExpect(t)
	mockMetric := getMockMetric()
	mockMetric.New("MockMetric")

	mockMetric.Inc("MockMetric")
	count, err := mockMetric.Get("MockMetric")
	expect.Nil(err)
	expect.Equal(int64(1), count)

	mockMetric.Dec("MockMetric")
	count, err = mockMetric.Get("MockMetric")
	expect.Nil(err)
	expect.Equal(int64(0), count)

}
