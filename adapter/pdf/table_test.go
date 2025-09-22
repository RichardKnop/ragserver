package pdf

import (
	"bytes"
	_ "embed"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"go.uber.org/zap"
)

//go:embed testdata/tables.html
var TestTables string

func TestParseTables(t *testing.T) {
	t.Parallel()

	tables, err := parseTables(zap.NewNop(), bytes.NewBufferString(TestTables))
	require.NoError(t, err)

	expected := []Table{
		{
			Rows: []Row{
				{
					"Scope 1 and Scope 2 (location & market based) Emissions (MTCO2e)",
					"2022",
				},
				{
					"Total Scope 1",
					"77,476",
				},
				{
					"Total Scope 2 (location)",
					"593,495",
				},
				{
					"Total Scope 2 (market)²",
					"4,424",
				},
				{
					"Total Scope 1 and 2 (location)",
					"670,972",
				},
				{
					"Total Scope 1 and 2 (market)",
					"81,901",
				},
				{
					"Carbon offsets purchased",
					"82,414",
				},
				{
					"Remaining Scope 1 and 2 (market)*",
					"",
				},
			},
		},
		{
			Rows: []Row{
				{
					"Scope 3 emissions (MTCO2e)",
					"2022",
				},
				{
					"Category 1: Purchased goods and services",
					"1,300,698",
				},
				{
					"Category 2: Capital goods",
					"293,289",
				},
				{
					"Category 3: Fuel and energy-related activities (not included in Scope 1 or 2)",
					"123,938",
				},
				{
					"Category 5: Waste generated in operations",
					"12,730",
				},
				{
					"Category 6: Employee business travel (air travel only)",
					"27,403",
				},
				{
					"Category 7: Employee commuting (excluding remote work)",
					"289,051",
				},
			},
		},
		{
			Title: "Greenhouse Gas Emissions Factors and Sources: Scope 1 and 2",
			Rows: []Row{
				{
					"Scope",
					"Category",
					"Emissions Source(s)",
					"Emissions Factor Employed",
				},
				{
					"Scope 1",
					"Stationary",
					"Diesel Natural gas Propane Fuel oil ##2",
					"Environmental Protection Agency's Center for Corporate Climate Leadership Emission Factors for GHG\n                Inventories hub (March 2023)",
				},
				{
					"Scope 1",
					"Mobile",
					"Diesel Jet fuel",
					"Environmental Protection Agency's Center for Corporate Climate Leadership Emission Factors for GHG\n                Inventories hub",
				},
				{
					"Scope 1",
					"Fugitive Emissions",
					"Fire suppressant Refrigerant",
					"(March 2023) Environmental Protection Agency's Center for Corporate Climate Leadership Emission Factors\n                for GHG Inventories hub (March 2023)",
				},
				{
					"Scope 1",
					"Location-based",
					"Purchased electricity Renewable power - on-site Purchased steam Chilled water",
					"U.S.: U.S. EPA Emissions & Generation Resource Integrated Database (eGRID) 2021. Data sources are pulled\n                at the eGRID subregion level. Canada: Environment Canada 2021 National Inventory Report (2019 data) All\n                Other Countries: Intemational Energy Agency (IEA) CO2 Emissions from Fuel",
				},
				{
					"",
					"Market-based",
					"Purchased electricity Renewable power - on-site Purchased steam Chilled water Power Purchase Agreements\n                Energy Attribute Certificates",
					"Combustion 2021 version. Wells Fargo applies the hierarchy from the GHG Protocol Scope 2 Guidance: 1.\n                Energy attribute certificates or equivalent instruments (RECs) Purchase Agreements (PPAs) 2. Contracts\n                for electricity, such as Power 3. Supplier/Utility emission rates 5. Other grid-average emissions\n                factors (ir 4. Residual mix (sub-national or national) methodology) accordance with the location-based\n                100% renewable energy from Energy Where possible, Wells Fargo consumes",
				},
			},
		},
	}
	assert.Equal(t, expected, tables)
}

func TestTable_ToContexts(t *testing.T) {
	t.Parallel()

	t.Run("No rows", func(t *testing.T) {
		aTable := Table{}
		contexts := aTable.ToContexts()
		assert.Empty(t, contexts)
	})

	t.Run("Two column table", func(t *testing.T) {
		aTable := Table{
			Rows: []Row{
				{
					"Scope 1 and Scope 2 (location & market based) Emissions (MTCO2e)",
					"2022",
				},
				{
					"Total Scope 1",
					"77,476",
				},
				{
					"Total Scope 2 (location)",
					"593,495",
				},
				{
					"Total Scope 2 (market)²",
					"4,424",
				},
				{
					"Total Scope 1 and 2 (location)",
					"670,972",
				},
				{
					"Total Scope 1 and 2 (market)",
					"81,901",
				},
				{
					"Carbon offsets purchased",
					"82,414",
				},
				{
					"Remaining Scope 1 and 2 (market)*",
					"",
				},
			},
		}
		contexts := aTable.ToContexts()
		expected := []string{
			"Total Scope 1: For year 2022: 77,476",
			"Total Scope 2 (location): For year 2022: 593,495",
			"Total Scope 2 (market)²: For year 2022: 4,424",
			"Total Scope 1 and 2 (location): For year 2022: 670,972",
			"Total Scope 1 and 2 (market): For year 2022: 81,901",
			"Carbon offsets purchased: For year 2022: 82,414",
		}
		assert.Equal(t, expected, contexts)
	})
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
