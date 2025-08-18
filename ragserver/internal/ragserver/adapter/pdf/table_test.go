package pdf

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
				Name: "Total Scope 1",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2019, Number: NewNumber(86602, "86,602")},
					{Year: 2020, Number: NewNumber(78087, "78,087")},
					{Year: 2021, Number: NewNumber(73319, "73,319")},
					{Year: 2022, Number: NewNumber(77476, "77,476")},
				},
			},
			{
				Name: "Total Scope 2 (location)",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2019, Number: NewNumber(771327, "771,327")},
					{Year: 2020, Number: NewNumber(694011, "694,011")},
					{Year: 2021, Number: NewNumber(569633, "569,633")},
					{Year: 2022, Number: NewNumber(593495, "593,495")},
				},
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

	tables, err := NewTables(testData)
	require.NoError(t, err)
	require.Len(t, tables, 1)
	actual := tables[0]
	expected := Table{
		Title: "Scope 3 emissions",
		Rows: []TableRow{
			{
				Name: "Category 1: Purchased goods and services",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2020, Number: NewNumber(1639281, "1,639,281")},
					{Year: 2021, Number: NewNumber(1429619, "1,429,619")},
					{Year: 2022, Number: NewNumber(1300698, "1,300,698")},
				},
			},
			{
				Name: "Category 2: Capital goods",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2020, Number: NewNumber(358268, "358,268")},
					{Year: 2021, Number: NewNumber(348249, "348,249")},
					{Year: 2022, Number: NewNumber(293289, "293,289")},
				},
			},
			{
				Name: "Category 3: Fuel and energy-related activities (not included in Scope 1 or 2)",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2020, Number: NewNumber(123970, "123,970")},
					{Year: 2021, Number: NewNumber(121357, "121,357")},
					{Year: 2022, Number: NewNumber(123938, "123,938")},
				},
			},
			{
				Name: "Category 5: Waste generated in operations",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2020, Number: NewNumber(7622, "7,622")},
					{Year: 2021, Number: NewNumber(13058, "13,058")},
					{Year: 2022, Number: NewNumber(12730, "12,730")},
				},
			},
			{
				Name: "Category 6: Employee business travel (air travel only)",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2020, Number: NewNumber(14111, "14,111")},
					{Year: 2021, Number: NewNumber(4795, "4,795")},
					{Year: 2022, Number: NewNumber(27403, "27,403")},
				},
			},
			{
				Name: "Category 7: Employee commuting (excluding remote work)",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2020, Number: NewNumber(313757, "313,757")},
					{Year: 2021, Number: NewNumber(218795, "218,795")},
					{Year: 2022, Number: NewNumber(289051, "289,051")},
				},
			},
		},
	}
	assert.Equal(t, expected.Title, actual.Title)
	assert.Equal(t, expected.Rows, actual.Rows)
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

	tables, err := NewTables(testData)
	require.NoError(t, err)
	require.Len(t, tables, 1)
	actual := tables[0]
	expected := Table{
		Title: "Overview of renewable energy activities in operations",
		Rows: []TableRow{
			{
				Name: "Total electricity consumed",
				Unit: "MWh",
				YearNumbers: []YearNumber{
					{Year: 2020, Number: NewNumber(1654354, "1,654,354")},
					{Year: 2021, Number: NewNumber(1550417, "1,550,417")},
					{Year: 2022, Number: NewNumber(1579854, "1,579,854")},
				},
			},
			{
				Name: "Total renewable energy purchased",
				Unit: "MWh",
				YearNumbers: []YearNumber{
					{Year: 2020, Number: NewNumber(1666777, "1,666,777")},
					{Year: 2021, Number: NewNumber(1673872, "1,673,872")},
					{Year: 2022, Number: NewNumber(1584509, "1,584,509")},
				},
			},
			{
				Name: "Renewable energy % of total electricity use",
				Unit: "%",
				YearNumbers: []YearNumber{
					{Year: 2020, Number: NewNumber(101, "101")},
					{Year: 2021, Number: NewNumber(108, "108")},
					{Year: 2022, Number: NewNumber(100, "100")},
				},
			},
			{
				Name: "Total capacity from long-term agreements supporting new sources of renewable energy",
				Unit: "MW",
				YearNumbers: []YearNumber{
					{Year: 2020, Number: NewNumber(186, "186")},
					{Year: 2021, Number: NewNumber(210, "210")},
					{Year: 2022, Number: NewNumber(210, "210")},
				},
			},
		},
	}
	assert.Equal(t, expected.Title, actual.Title)
	assert.Equal(t, expected.Rows, actual.Rows)
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

	tables, err := NewTables(testData)
	require.NoError(t, err)
	require.Len(t, tables, 1)
	actual := tables[0]
	expected := Table{
		Title: "Scope 1 and Scope 2 emissions (location and market based)",
		Rows: []TableRow{
			{
				Name: "Total Scope 1",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2019, Number: NewNumber(86602, "86,602")},
					{Year: 2020, Number: NewNumber(78087, "78,087")},
					{Year: 2021, Number: NewNumber(73319, "73,319*")},
					{Year: 2022, Number: NewNumber(77476, "77,476*")},
				},
			},
			{
				Name: "Total Scope 2 (location)",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2019, Number: NewNumber(771327, "771,327")},
					{Year: 2020, Number: NewNumber(694011, "694,011")},
					{Year: 2021, Number: NewNumber(569633, "569,633*")},
					{Year: 2022, Number: NewNumber(593495, "593,495*")},
				},
			},
			{
				Name: "Total Scope 2 (market)",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2019, Number: NewNumber(4988, "4,988")},
					{Year: 2020, Number: NewNumber(3614, "3,614")},
					{Year: 2021, Number: NewNumber(1792, "1,792*")},
					{Year: 2022, Number: NewNumber(4424, "4,424*")},
				},
			},
			{
				Name: "Total Scope 1 and 2 (location)",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2019, Number: NewNumber(857929, "857,929")},
					{Year: 2020, Number: NewNumber(772098, "772,098")},
					{Year: 2021, Number: NewNumber(642952, "642,952*")},
					{Year: 2022, Number: NewNumber(670972, "670,972*")},
				},
			},
			{
				Name: "Total Scope 1 and 2 (market)",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2019, Number: NewNumber(91591, "91,591")},
					{Year: 2020, Number: NewNumber(81701, "81,701")},
					{Year: 2021, Number: NewNumber(75111, "75,111*")},
					{Year: 2022, Number: NewNumber(81901, "81,901*")},
				},
			},
			{
				Name: "Carbon offsets purchased",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2019, Number: NewNumber(98981, "98,981")},
					{Year: 2020, Number: NewNumber(92019, "92,019")},
					{Year: 2021, Number: NewNumber(81809, "81,809*")},
					{Year: 2022, Number: NewNumber(82414, "82,414*")},
				},
			},
			{
				Name: "Remaining Scope 1 and 2 (market)",
				Unit: "MTCO2e",
				YearNumbers: []YearNumber{
					{Year: 2019, Number: NewNumber(0, "0")},
					{Year: 2020, Number: NewNumber(0, "0")},
					{Year: 2021, Number: NewNumber(0, "0*")},
					{Year: 2022, Number: NewNumber(0, "0*")},
				},
			},
			{
				Name: "Reduction in total Scope 1 and 2 (location) GHG emissions (from 2019 baseline)",
				Unit: "%",
				YearNumbers: []YearNumber{
					{Year: 2019, Number: Number{}},
					{Year: 2020, Number: NewNumber(10, "10")},
					{Year: 2021, Number: NewNumber(25, "25")},
					{Year: 2022, Number: NewNumber(22, "22")},
				},
			},
		},
	}
	assert.Equal(t, expected.Title, actual.Title)
	assert.Equal(t, expected.Rows, actual.Rows)
}

func TestNewTable_Simple_Example1(t *testing.T) {
	t.Parallel()

	testData := `
Statement of Greenhouse Gas Emissions 
For the year ended December 31, 2022 
Greenhouse Gas Emissions 
Scope 1 and Scope 2 (location & market based) Emissions (MTCO2e)
1
2022 
Total Scope 1 
77,476 
Total Scope 2 (location) 
593,495 
Total Scope 2 (market)
2
4,424 
Total Scope 1 and 2 (location) 
670,972 
Total Scope 1 and 2 (market) 
81,901 
Carbon offsets purchased
3
82,414 
Remaining Scope 1 and 2 (market)
4
0 
.
	`

	tables, err := NewTables(testData)
	require.NoError(t, err)
	require.Len(t, tables, 1)
	actual := tables[0]
	expected := Table{
		Title: "Scope 1 and Scope 2 (location & market based) Emissions (MTCO2e)",
		Rows: []TableRow{
			{
				Name: "Total Scope 1",
				YearNumbers: []YearNumber{
					{Year: 2022, Number: NewNumber(77476, "77,476")},
				},
			},
			{
				Name: "Total Scope 2 (location)",
				YearNumbers: []YearNumber{
					{Year: 2022, Number: NewNumber(593495, "593,495")},
				},
			},
			{
				Name: "Total Scope 2 (market)",
				YearNumbers: []YearNumber{
					{Year: 2022, Number: NewNumber(4424, "4,424")},
				},
			},
			{
				Name: "Total Scope 1 and 2 (location)",
				YearNumbers: []YearNumber{
					{Year: 2022, Number: NewNumber(670972, "670,972")},
				},
			},
			{
				Name: "Total Scope 1 and 2 (market)",
				YearNumbers: []YearNumber{
					{Year: 2022, Number: NewNumber(81901, "81,901")},
				},
			},
			{
				Name: "Carbon offsets purchased",
				YearNumbers: []YearNumber{
					{Year: 2022, Number: NewNumber(82414, "82,414")},
				},
			},
			{
				Name: "Remaining Scope 1 and 2 (market)",
				YearNumbers: []YearNumber{
					{Year: 2022, Number: NewNumber(0, "0")},
				},
			},
		},
	}
	assert.Equal(t, expected.Title, actual.Title)
	assert.Equal(t, expected.Rows, actual.Rows)
}

func TestNewTable_Simple_TwoTables(t *testing.T) {
	t.Parallel()
	// t.Skip()

	testData := `
Statement of Greenhouse Gas Emissions 
For the year ended December 31, 2022 
Greenhouse Gas Emissions 
Scope 1 and Scope 2 (location & market based) Emissions (MTCO2e)
1
2022 
Total Scope 1 
77,476 
Total Scope 2 (location) 
593,495 
Total Scope 2 (market)
2
4,424 
Total Scope 1 and 2 (location) 
670,972 
Total Scope 1 and 2 (market) 
81,901 
Carbon offsets purchased
3
82,414 
Remaining Scope 1 and 2 (market)
4
0 
Scope 3 emissions (MTCO2e) 
2022 
Category 1: Purchased goods and services 
1,300,698 
Category 2: Capital goods 
293,289 
Category 3: Fuel and energy-related activities (not included in Scope 1 or 2) 
123,938 
Category 5: Waste generated in operations 
12,730 
Category 6: Employee business travel (air travel only) 
27,403 
Category 7: Employee commuting (excluding remote work) 
289,051 
The accompanying notes are an integral part of the Statement of Greenhouse Gas Emissions
. 
1
 MTCO2e stands for metric tons of carbon dioxide equivalent. 
2
 A location-based method reflects the average emissions intensity of grids on which energy consumption occurs (using grid average emission factor data). A market-based 
method reflects emissions from electricity that Wells Fargo has purposefully chosen. It derives emission factors from contractual instruments, which include any type of 
contract between two parties for the sale and purchase of energy bundled with attributes about the energy generation, or for unbundled attribute claims.
3
 In 2022, Wells Fargo purchased carbon offsets from projects that remove and store carbon. A portion of these credits are Verified Carbon Standard (VSC) certified and 
have also achieved the add-on Climate, Community and Biodiversity (CCB) certification and are therefore VSC+CCB certified. The remaining portion is certified by the 
Climate Action Reserve (CAR).
4
 As part of its journey toward net zero, Wells Fargo has implemented carbon reduction strategies and purchased energy attribute certificates and carbon offsets sufficient 
to cover its total Scope 1 and Scope 2 (market-based) emissions for 2022. 
	`

	tables, err := NewTables(testData)
	require.NoError(t, err)
	require.Len(t, tables, 2)

	actual := tables[0]
	expected := Table{
		Title: "Scope 1 and Scope 2 (location & market based) Emissions (MTCO2e)",
		Rows: []TableRow{
			{
				Name:        "Total Scope 1",
				YearNumbers: []YearNumber{{Year: 2022, Number: NewNumber(77476, "77,476")}},
			},
			{
				Name:        "Total Scope 2 (location)",
				YearNumbers: []YearNumber{{Year: 2022, Number: NewNumber(593495, "593,495")}},
			},
			{
				Name:        "Total Scope 2 (market)",
				YearNumbers: []YearNumber{{Year: 2022, Number: NewNumber(4424, "4,424")}},
			},
			{
				Name:        "Total Scope 1 and 2 (location)",
				YearNumbers: []YearNumber{{Year: 2022, Number: NewNumber(670972, "670,972")}},
			},
			{
				Name:        "Total Scope 1 and 2 (market)",
				YearNumbers: []YearNumber{{Year: 2022, Number: NewNumber(81901, "81,901")}},
			},
			{
				Name:        "Carbon offsets purchased",
				YearNumbers: []YearNumber{{Year: 2022, Number: NewNumber(82414, "82,414")}},
			},
			{
				Name:        "Remaining Scope 1 and 2 (market)",
				YearNumbers: []YearNumber{{Year: 2022, Number: NewNumber(0, "0")}},
			},
		},
	}
	assert.Equal(t, expected.Title, actual.Title)
	assert.Equal(t, expected.Rows, actual.Rows)

	actual = tables[1]
	expected = Table{
		Title: "Scope 3 emissions (MTCO2e)",
		Rows: []TableRow{
			{
				Name:        "Category 1: Purchased goods and services",
				YearNumbers: []YearNumber{{Year: 2022, Number: NewNumber(1300698, "1,300,698")}},
			},
			{
				Name:        "Category 2: Capital goods",
				YearNumbers: []YearNumber{{Year: 2022, Number: NewNumber(293289, "293,289")}},
			},
			{
				Name:        "Category 3: Fuel and energy-related activities (not included in Scope 1 or 2)",
				YearNumbers: []YearNumber{{Year: 2022, Number: NewNumber(123938, "123,938")}},
			},
			{
				Name:        "Category 5: Waste generated in operations",
				YearNumbers: []YearNumber{{Year: 2022, Number: NewNumber(12730, "12,730")}},
			},
			{
				Name:        "Category 6: Employee business travel (air travel only)",
				YearNumbers: []YearNumber{{Year: 2022, Number: NewNumber(27403, "27,403")}},
			},
			{
				Name:        "Category 7: Employee commuting (excluding remote work)",
				YearNumbers: []YearNumber{{Year: 2022, Number: NewNumber(289051, "289,051")}},
			},
		},
	}
	assert.Equal(t, expected.Title, actual.Title)
	assert.Equal(t, expected.Rows, actual.Rows)
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
		{"123", NewNumber(123, "123"), true},
		{"123.45", NewNumber(123.45, "123.45"), true},
		{"abc", Number{}, false},
		{"123abc", Number{}, false},
		{"1,666,777", NewNumber(1666777, "1,666,777"), true},
		{"14,111", NewNumber(14111, "14,111"), true},
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
		{"123", NewNumber(123, "123"), true},
		{"123.45", NewNumber(123.45, "123.45"), true},
		{"abc", Number{}, false},
		{"123abc", Number{}, false},
		{"1,666,777", NewNumber(1666777, "1,666,777"), true},
		{"14,111", NewNumber(14111, "14,111"), true},
		{"2020", NewNumber(2020, "2020"), true},
		{"2020*", NewNumber(2020, "2020*"), true},
		{"2019 (baseline)", NewNumber(2019, "2019 (baseline)"), true},
	}

	for _, test := range tests {
		actual, ok := isNumber(test.input)
		assert.Equal(t, test.expected, actual)
		assert.Equal(t, test.ok, ok)
	}
}

func TestNumber_ValidYear(t *testing.T) {
	t.Parallel()

	tests := []struct {
		input    Number
		expected bool
	}{
		{NewNumber(1850, "1850"), false},
		{NewNumber(1999, "1999"), true},
		{NewNumber(2020, "2020"), true},
		{NewNumber(2150, "2150"), false},
	}

	for _, test := range tests {
		actual := test.input.ValidYear()
		assert.Equal(t, test.expected, actual)
	}
}
