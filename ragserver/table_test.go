package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTable_ToContext(t *testing.T) {
	t.Parallel()

	table := Table{
		Title: "Scope 1 and Scope 2 emissions (location and market based)",
		Rows: []TableRow{
			{
				Name:        "Total Scope 1",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2019, Number: NewNumber(86602)}, {Year: 2020, Number: NewNumber(78087)}, {Year: 2021, Number: NewNumber(73319)}, {Year: 2022, Number: NewNumber(77476)}},
			},
			{
				Name:        "Total Scope 2 (location)",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2019, Number: NewNumber(771327)}, {Year: 2020, Number: NewNumber(694011)}, {Year: 2021, Number: NewNumber(569633)}, {Year: 2022, Number: NewNumber(593495)}},
			},
		},
	}

	expected := []string{
		"Scope 1 and Scope 2 emissions (location and market based): Total Scope 1 for year 2019 is 86602 MTCO2e",
		"Scope 1 and Scope 2 emissions (location and market based): Total Scope 1 for year 2020 is 78087 MTCO2e",
		"Scope 1 and Scope 2 emissions (location and market based): Total Scope 1 for year 2021 is 73319 MTCO2e",
		"Scope 1 and Scope 2 emissions (location and market based): Total Scope 1 for year 2022 is 77476 MTCO2e",
		"Scope 1 and Scope 2 emissions (location and market based): Total Scope 2 (location) for year 2019 is 771327 MTCO2e",
		"Scope 1 and Scope 2 emissions (location and market based): Total Scope 2 (location) for year 2020 is 694011 MTCO2e",
		"Scope 1 and Scope 2 emissions (location and market based): Total Scope 2 (location) for year 2021 is 569633 MTCO2e",
		"Scope 1 and Scope 2 emissions (location and market based): Total Scope 2 (location) for year 2022 is 593495 MTCO2e",
	}

	actual := table.ToContexts()
	assert.Equal(t, expected, actual)
}

func TestNewTable_Example1(t *testing.T) {
	t.Parallel()

	testData := `
45 
Wells Fargo | 2022 TCFD Report 
Governance 
Strategy 
Risk management 
Metrics and targets 
Aside from efforts to reduce energy consumption in our buildings, we look for opportunities to reduce 
emissions across all calculated areas of our business operations, including through collaborative engagement 
with 
suppliers. 
Scope 3 emissions
23
Unit
24
2020 
2021* 
2022* 
Category 1: Purchased goods and services 
MTCO2e 
1,639,281 
1,429,619 
1,300,698 
Category 2: Capital goods 
MTCO2e 
358,268 
348,249 
293,289 
Category 3: Fuel and energy-related activities (not included in 
Scope 1 or 2) 
MTCO2e 
123,970 
121,357 
123,938 
Category 5: Waste generated in operations 
MTCO2e 
7,622 
13,058 
12,730 
Category 6: Employee business travel (air travel only) 
MTCO2e 
14,111 
4,795 
27,403 
Category 7: Employee commuting (excluding remote work) 
MTCO2e 
313,757 
218,795 
289,051 
*Wells Fargo's Statement of Greenhouse Gas Emissions, which can be found on our 
Goals and Reporting website
, has been reviewed by an 
independent accountant for the years ended December 31, 2021, and 2022.
23
 This report includes relevant Scope 3 categories for which Wells Fargo had calculated emissions for the year ended 2022. 
24
 MTCO2e stands for metric tons of carbon dioxide equivalent. 
	`

	actual, err := NewTable(testData)
	require.NoError(t, err)
	expected := Table{
		Title: "Scope 3 emissions",
		Rows: []TableRow{
			{
				Name:        "Category 1: Purchased goods and services",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2020, Number: NewNumber(1639281)}, {Year: 2021, Number: NewNumber(1429619)}, {Year: 2022, Number: NewNumber(1300698)}},
			},
			{
				Name:        "Category 2: Capital goods",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2020, Number: NewNumber(358268)}, {Year: 2021, Number: NewNumber(348249)}, {Year: 2022, Number: NewNumber(293289)}},
			},
			{
				Name:        "Category 3: Fuel and energy-related activities (not included in Scope 1 or 2)",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2020, Number: NewNumber(123970)}, {Year: 2021, Number: NewNumber(121357)}, {Year: 2022, Number: NewNumber(123938)}},
			},
			{
				Name:        "Category 5: Waste generated in operations",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2020, Number: NewNumber(7622)}, {Year: 2021, Number: NewNumber(13058)}, {Year: 2022, Number: NewNumber(12730)}},
			},
			{
				Name:        "Category 6: Employee business travel (air travel only)",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2020, Number: NewNumber(14111)}, {Year: 2021, Number: NewNumber(4795)}, {Year: 2022, Number: NewNumber(27403)}},
			},
			{
				Name:        "Category 7: Employee commuting (excluding remote work)",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2020, Number: NewNumber(313757)}, {Year: 2021, Number: NewNumber(218795)}, {Year: 2022, Number: NewNumber(289051)}},
			},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestNewTable_Example2(t *testing.T) {
	t.Parallel()

	testData := `
44 
Wells Fargo | 2022 TCFD Report 
Governance 
Strategy 
Risk management 
Metrics and targets 
Wells Fargo operational sustainability highlights in renewable energy: 
Overview of renewable energy activities in operations 
Unit 
2020 
2021 
2022 
Total electricity consumed
19
MWh 
1,654,354 
1,550,417 
1,579,854 
Total renewable energy purchased
20
MWh 
1,666,777 
1,673,872 
1,584,509 
Renewable energy % of total electricity use
21
% 
101 
108 
100 
Total capacity from long-term agreements supporting new sources of 
renewable energy
22
MW 
186 
210 
210 
19
 Includes purchased electricity and self-supplied electricity generated through Wells Fargo's on-site solar program. 
20
 Total renewable energy purchased includes self-supply renewable energy where Wells Fargo generates renewable energy from on-site solar installations, power purchase agreements, which are contracts for the purchase of power and 
associated Renewable Energy Certificates, as well as Unbundled Renewable Energy Certificates, which are sold, delivered, or purchased separately from the electricity generated from the renewable resource.
21
 Wells Fargo secures enough Renewable Energy Certificates to meet or exceed our annual consumption of purchased electricity. 
22
 New sources of renewable energy are defined as assets where commercial operation was achieved no earlier than 12 months prior to contract execution. This data includes cumulative new renewable energy generation capacity contracted 
by Wells Fargo. Some assets have not yet achieved commercial operation and are under construction. 
	`

	actual, err := NewTable(testData)
	require.NoError(t, err)
	expected := Table{
		Title: "Overview of renewable energy activities in operations",
		Rows: []TableRow{
			{
				Name:        "Total electricity consumed",
				Unit:        "MWh",
				YearNumbers: []YearNumber{{Year: 2020, Number: NewNumber(1654354)}, {Year: 2021, Number: NewNumber(1550417)}, {Year: 2022, Number: NewNumber(1579854)}},
			},
			{
				Name:        "Total renewable energy purchased",
				Unit:        "MWh",
				YearNumbers: []YearNumber{{Year: 2020, Number: NewNumber(1666777)}, {Year: 2021, Number: NewNumber(1673872)}, {Year: 2022, Number: NewNumber(1584509)}},
			},
			{
				Name:        "Renewable energy % of total electricity use",
				Unit:        "%",
				YearNumbers: []YearNumber{{Year: 2020, Number: NewNumber(101)}, {Year: 2021, Number: NewNumber(108)}, {Year: 2022, Number: NewNumber(100)}},
			},
			{
				Name:        "Total capacity from long-term agreements supporting new sources of renewable energy",
				Unit:        "MW",
				YearNumbers: []YearNumber{{Year: 2020, Number: NewNumber(186)}, {Year: 2021, Number: NewNumber(210)}, {Year: 2022, Number: NewNumber(210)}},
			},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestNewTable_Example3(t *testing.T) {
	t.Parallel()

	testData := `
43 
Wells Fargo | 2022 TCFD Report 
Governance 
Strategy 
Risk management 
Metrics and targets 
Scope 1 and Scope 2 emissions (location and market based)
13
,14
Unit
15
2019 (baseline) 
2020 
2021 
2022 
Total Scope 1 
MTCO2e 
86,602 
78,087 
73,319* 
77,476* 
Total Scope 2 (location) 
MTCO2e 
771,327 
694,011 
569,633* 
593,495* 
Total Scope 2 (market)
16
MTCO2e 
4,988 
3,614 
1,792* 
4,424* 
Total Scope 1 and 2 (location) 
MTCO2e 
857,929 
772,098 
642,952* 
670,972* 
Total Scope 1 and 2 (market) 
MTCO2e 
91,591 
81,701 
75,111* 
81,901* 
Carbon offsets purchased
17
MTCO2e 
98,981 
92,019 
81,809* 
82,414* 
Remaining Scope 1 and 2 (market)
18
MTCO2e 
0 
0 
0* 
0* 
Reduction in total Scope 1 and 2 (location) GHG emissions 
(from 2019 baseline) 
% 
— 
10 
25 
22 
*Wells Fargo's Statement of Greenhouse Gas Emissions, which can be found on our
 Goals and Reporting website
, has been reviewed by an 
independent accountant for the years ended December 31, 2021, and 2022.
	`

	actual, err := NewTable(testData)
	require.NoError(t, err)
	expected := Table{
		Title: "Scope 1 and Scope 2 emissions (location and market based)",
		Rows: []TableRow{
			{
				Name:        "Total Scope 1",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2019, Number: NewNumber(86602)}, {Year: 2020, Number: NewNumber(78087)}, {Year: 2021, Number: NewNumber(73319)}, {Year: 2022, Number: NewNumber(77476)}},
			},
			{
				Name:        "Total Scope 2 (location)",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2019, Number: NewNumber(771327)}, {Year: 2020, Number: NewNumber(694011)}, {Year: 2021, Number: NewNumber(569633)}, {Year: 2022, Number: NewNumber(593495)}},
			},
			{
				Name:        "Total Scope 2 (market)",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2019, Number: NewNumber(4988)}, {Year: 2020, Number: NewNumber(3614)}, {Year: 2021, Number: NewNumber(1792)}, {Year: 2022, Number: NewNumber(4424)}},
			},
			{
				Name:        "Total Scope 1 and 2 (location)",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2019, Number: NewNumber(857929)}, {Year: 2020, Number: NewNumber(772098)}, {Year: 2021, Number: NewNumber(642952)}, {Year: 2022, Number: NewNumber(670972)}},
			},
			{
				Name:        "Total Scope 1 and 2 (market)",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2019, Number: NewNumber(91591)}, {Year: 2020, Number: NewNumber(81701)}, {Year: 2021, Number: NewNumber(75111)}, {Year: 2022, Number: NewNumber(81901)}},
			},
			{
				Name:        "Carbon offsets purchased",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2019, Number: NewNumber(98981)}, {Year: 2020, Number: NewNumber(92019)}, {Year: 2021, Number: NewNumber(81809)}, {Year: 2022, Number: NewNumber(82414)}},
			},
			{
				Name:        "Remaining Scope 1 and 2 (market)",
				Unit:        "MTCO2e",
				YearNumbers: []YearNumber{{Year: 2019, Number: NewNumber(0)}, {Year: 2020, Number: NewNumber(0)}, {Year: 2021, Number: NewNumber(0)}, {Year: 2022, Number: NewNumber(0)}},
			},
			{
				Name:        "Reduction in total Scope 1 and 2 (location) GHG emissions (from 2019 baseline)",
				Unit:        "%",
				YearNumbers: []YearNumber{{Year: 2019, Number: Number{}}, {Year: 2020, Number: NewNumber(10)}, {Year: 2021, Number: NewNumber(25)}, {Year: 2022, Number: NewNumber(22)}},
			},
		},
	}
	assert.Equal(t, expected, actual)
}

func TestIsNumberOrNotAvailable(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected Number
		ok       bool
	}{
		{"%", Number{}, false},
		{"-—", Number{}, true},
		{"—", Number{}, true},
		{"-", Number{}, true},
		{"N/A", Number{}, true},
		{"123", NewNumber(123), true},
		{"123.45", NewNumber(123.45), true},
		{"abc", Number{}, false},
		{"123abc", Number{}, false},
		{"1,666,777", NewNumber(1666777), true},
		{"14,111", NewNumber(14111), true},
	}

	for _, test := range tests {
		actual, ok := isNumberOrNotAvailable(test.input)
		assert.Equal(t, test.expected, actual)
		assert.Equal(t, test.ok, ok)
	}
}

func TestIsNumber(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    string
		expected Number
		ok       bool
	}{
		{"123", NewNumber(123), true},
		{"123.45", NewNumber(123.45), true},
		{"abc", Number{}, false},
		{"123abc", Number{}, false},
		{"1,666,777", NewNumber(1666777), true},
		{"14,111", NewNumber(14111), true},
	}

	for _, test := range tests {
		actual, ok := isNumber(test.input)
		assert.Equal(t, test.expected, actual)
		assert.Equal(t, test.ok, ok)
	}
}
