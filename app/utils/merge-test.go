package utils

import (
	"encoding/json"

	"github.com/imdario/mergo"
	. "github.com/onsi/ginkgo"
	. "github.com/onsi/gomega"
)

type StarWarsHeroTest struct {
	FirstName string
	LastName  string
	Age       int
	Ship      *string
	IsJedi    bool
}

type StarWarsVillianTest struct {
	FirstName  string
	Ship       *string
	IsDarkJedi bool
}

var _ = Describe("Merge", func() {
	Context("Same Structures", func() {
		It("should overwrite all matching keys with with non-zero values", func() {
			xwing := "x-wing"
			destHero := StarWarsHeroTest{FirstName: "Luke", LastName: "Skywalker", Age: 25, Ship: &xwing, IsJedi: true}
			sourceHero := StarWarsHeroTest{FirstName: "Hans", LastName: "Solo", Age: 75}

			bytesDest, err := json.Marshal(destHero)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(bytesDest)).Should(Equal(`{"FirstName":"Luke","LastName":"Skywalker","Age":25,"Ship":"x-wing","IsJedi":true}`))

			bytesSource, err := json.Marshal(sourceHero)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(bytesSource)).Should(Equal(`{"FirstName":"Hans","LastName":"Solo","Age":75,"Ship":null,"IsJedi":false}`))

			var dataDest interface{}
			err = json.Unmarshal(bytesDest, &dataDest)
			Ω(err).ShouldNot(HaveOccurred())

			var dataSource interface{}
			err = json.Unmarshal(bytesSource, &dataSource)
			Ω(err).ShouldNot(HaveOccurred())

			m := dataDest.(map[string]interface{})
			err = mergo.MergeWithOverwrite(&m, dataSource)
			Ω(err).ShouldNot(HaveOccurred())

			bytesDest, err = json.Marshal(m)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(bytesDest)).Should(Equal(`{"Age":75,"FirstName":"Hans","IsJedi":false,"LastName":"Solo","Ship":"x-wing"}`))
		})
	})

	Context("Different Structures", func() {
		It("should overwrite all matching keys with with non-zero values", func() {
			xwing := "x-wing"
			destHero := StarWarsHeroTest{FirstName: "Luke", LastName: "Skywalker", Age: 25, Ship: &xwing, IsJedi: true}
			sourceVillian := StarWarsVillianTest{FirstName: "Darth", IsDarkJedi: true}

			bytesDest, err := json.Marshal(destHero)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(bytesDest)).Should(Equal(`{"FirstName":"Luke","LastName":"Skywalker","Age":25,"Ship":"x-wing","IsJedi":true}`))

			bytesSource, err := json.Marshal(sourceVillian)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(bytesSource)).Should(Equal(`{"FirstName":"Darth","Ship":null,"IsDarkJedi":true}`))

			var dataDest interface{}
			err = json.Unmarshal(bytesDest, &dataDest)
			Ω(err).ShouldNot(HaveOccurred())

			var dataSource interface{}
			err = json.Unmarshal(bytesSource, &dataSource)
			Ω(err).ShouldNot(HaveOccurred())

			m := dataDest.(map[string]interface{})
			// err = mergo.MergeWithOverwrite(&m, dataSource)
			err = mergo.MergeWithOverwrite(&m, dataSource)
			Ω(err).ShouldNot(HaveOccurred())

			bytesDest, err = json.Marshal(m)
			Ω(err).ShouldNot(HaveOccurred())
			Ω(string(bytesDest)).Should(Equal(`{"Age":25,"FirstName":"Darth","IsDarkJedi":true,"IsJedi":true,"LastName":"Skywalker","Ship":"x-wing"}`))
		})
	})
})
