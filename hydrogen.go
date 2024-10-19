package main

import (
	"fmt"
	"log"
)

func CalculateHydrogen(pressure float64, temperature float64, volumeUnits string, pressureUnits string) *HydrogenType {
	result := new(HydrogenType)
	result.Pressure, result.PressureText = CalculatePressure(pressure, pressureUnits)
	result.PressureUnits = getPressureLabel(pressureUnits)
	result.MaxPressure, _ = CalculatePressure(float64(currentSettings.MaxGasPressure), pressureUnits)
	result.Volume, result.VolumeText = CalculateVolume(pressure, temperature, volumeUnits)
	result.VolumeUnits = getVolumeLabel(volumeUnits)
	result.MaxVolume, _ = CalculateVolume(float64(currentSettings.MaxGasPressure), temperature, volumeUnits)
	return result
}

func CalculatePressure(bar float64, pressureUnits string) (float64, string) {
	switch pressureUnits {
	case "Pa":
		return bar * 100000, fmt.Sprintf("%.0f", bar*100000)
	case "kPa":
		return bar * 100, fmt.Sprintf("%.0f", bar*100)
	case "MPa":
		return bar * 0.1, fmt.Sprintf("%.3f", bar*0.1)
	case "hPa":
		return bar * 1000, fmt.Sprintf("%.0f", bar*1000)
	case "daPA":
		return bar * 10000, fmt.Sprintf("%.0f", bar*10000)
	case "nm2":
		return bar * 100000, fmt.Sprintf("%.0f", bar*100000)
	case "ncm2":
		return bar * 10, fmt.Sprintf("%.0f", bar*10)
	case "nmm2":
		return bar * 0.1, fmt.Sprintf("%.2f", bar*0.1)
	case "knm2":
		return bar * 100, fmt.Sprintf("%.0f", bar*100)
	case "mbar":
		return bar * 1000, fmt.Sprintf("%.0f", bar*1000)
	case "µbar":
		return bar * 1000000, fmt.Sprintf("%.0f", bar*1000)
	case "dcm2":
		return bar * 1000000, fmt.Sprintf("%.0f", bar*1000)
	case "kgm2":
		return bar * 10197.16213, fmt.Sprintf("%.0f", bar*10197.16213)
	case "kgcm2":
		return bar * 1.019716213, fmt.Sprintf("%.1f", bar*1.019716213)
	case "kgmm2":
		return bar * 0.0101971621, fmt.Sprintf("%.3f", bar*0.0101971621)
	case "gcm2":
		return bar * 1019.716213, fmt.Sprintf("%.0f", bar*1019.716213)
	case "tft2":
		return bar * 1.0442717117, fmt.Sprintf("%.1f", bar*1.0442717117)
	case "tin2":
		return bar * 0.0072518869, fmt.Sprintf("%.3f", bar*0.0072518869)
	case "ltft2":
		return bar * 0.9323854568, fmt.Sprintf("%.1f", bar*0.9323854568)
	case "ltin2":
		return bar * 0.006474899, fmt.Sprintf("%.3f", bar*0.006474899)
	case "ksi":
		return bar * 0.0145037738, fmt.Sprintf("%.2f", bar*0.0145037738)
	case "psi":
		return bar * 14.503773773, fmt.Sprintf("%.0f", bar*14.50377373)
	case "psf":
		return bar * 2088.5434233, fmt.Sprintf("%.0f", bar*2088.5434233)
	case "torr":
		return bar * 750.0616827, fmt.Sprintf("%.0f", bar*750.0616827)
	case "cmhg":
		return bar * 75.006375542, fmt.Sprintf("%.0f", bar*75.00637554)
	case "mmhg":
		return bar * 750.06375542, fmt.Sprintf("%.0f", bar*750.06375542)
	case "inhg32":
		return bar * 29.530058647, fmt.Sprintf("%.1f", bar*29.530058647)
	case "inhg60":
		return bar * 29.613397101, fmt.Sprintf("%.1f", bar*29.613397101)
	case "cmh2o":
		return bar * 1019.7442889, fmt.Sprintf("%.0f", bar*1019.7442889)
	case "mmh2o":
		return bar * 10197.442889, fmt.Sprintf("%.0f", bar*10197.442889)
	case "inaq4":
		return bar * 401.47421331, fmt.Sprintf("%.0f", bar*401.47421331)
	case "ftaq4":
		return bar * 33.456229215, fmt.Sprintf("%.1f", bar*33.456229215)
	case "inaq60":
		return bar * 401.85980719, fmt.Sprintf("%.0f", bar*401.85980719)
	case "ftaq60":
		return bar * 33.488317266, fmt.Sprintf("%.1f", bar*33.488317266)
	case "atm":
		return bar * 1.019716213, fmt.Sprintf("%.2f", bar*1.019716213)
	default:
		return bar, fmt.Sprintf("%.2f", bar)
	}
}

// getPressureLabel returns the text label to use for the select pressure units
func getPressureLabel(units string) string {
	switch units {
	case "Pa":
		return "Pa"
	case "kPa":
		return "kPa"
	case "EPa":
		return "EPa"
	case "PPa":
		return "PPa"
	case "TPa":
		return "TPa"
	case "GPa":
		return "GPa"
	case "MPa":
		return "MPa"
	case "hPa":
		return "hPa"
	case "daPA":
		return "daPa"
	case "dPa":
		return "dPa"
	case "cPa":
		return "cPa"
	case "mPa":
		return "mPa"
	case "µPa":
		return "µPa"
	case "nPa":
		return "nPa"
	case "pPa":
		return "pPa"
	case "fPa":
		return "fPa"
	case "aPa":
		return "aPa"
	case "nm2":
		return "n/m2"
	case "ncm2":
		return "n/cm2"
	case "nmm2":
		return "n/mm2"
	case "knm2":
		return "kn/m2"
	case "mbar":
		return "mBar"
	case "µbar":
		return "µbar"
	case "dcm2":
		return "dc/sqm"
	case "kgm2":
		return "kg/sqm"
	case "kgcm2":
		return "kg/sqcm"
	case "kgmm2":
		return "kg/sqmm"
	case "gcm2":
		return "g/sqcm"
	case "tft2":
		return "t/sqft"
	case "tin2":
		return "t/sqin"
	case "ltft2":
		return "t/sqft"
	case "ltin2":
		return "t/sqin"
	case "ksi":
		return "ksi"
	case "psi":
		return "psi"
	case "psf":
		return "psf"
	case "torr":
		return "torr"
	case "cmhg":
		return "cm-hg"
	case "mmhg":
		return "mm-hg"
	case "inhg32":
		return "in-hg"
	case "inhg60":
		return "in-hg"
	case "cmh2o":
		return "cm-H2O"
	case "mmh2o":
		return "mm-H2O"
	case "inaq4":
		return "in-H2O"
	case "ftaq4":
		return "ft-H2O"
	case "inaq60":
		return "in-H2O"
	case "ftaq60":
		return "ft-H2O"
	case "atm":
		return "Atmosphere"
	default:
		return "Bar"
	}
}

const R = 8.31446261815324 // Ideal gas constant
// Ideal gas law : PV = nRT	where P = Pascal, V = cubic metres, n = moles, T = Kelvin

const GramsPerMol = 2.01588
const LitresPerMol = 22.71

// CalculateVolume returns the total volume of gas contained in the given units based on temperature and pressure using the system capacity in litres
func CalculateVolume(bar float64, temperature float64, volumeUnits string) (float64, string) {
	temperature = 273.15 + temperature                    // convert from Celsius to Kelvin
	pressure := bar * 100000                              // convert from Bar to Pascal
	volume := float64(currentSettings.GasCapacity) / 1000 // convert from litres to cubic metres
	mols := (pressure * volume) / (temperature * R)
	grams := mols * GramsPerMol
	NormalLitres := mols * LitresPerMol
	// Energy density of H2 = 0.0794444444kWh / mol
	//	kWh := mols * 0.0671288 * float64(currentSettings.FuelCellSettings.Efficiency) // H2 energy = 0.0671288 kWh/mole.
	kWh := mols * GramsPerMol * 0.000333 * float64(currentSettings.FuelCellSettings.Efficiency)

	//litres = val * 23.6422; // Litres per mol
	//let kg = val / 496.0613
	//let kwhr = val / 20308.6498
	//let retval = {
	//val: 0.0,
	//	label: "litres"
	//}
	switch volumeUnits {
	case "kg":
		kg := grams / 1000
		return kg, fmt.Sprintf("%.3f", kg)
	case "g":
		return grams, fmt.Sprintf("%d", int64(grams))
	case "litres":
		litres := mols * 22.71
		return litres, fmt.Sprintf("%d", int64(litres))
	case "cucm":
		cucm := mols * (LitresPerMol * 1000)
		return cucm, fmt.Sprintf("%d", int64(cucm))
	case "cum":
		cum := mols * (LitresPerMol * 0.001)
		return cum, fmt.Sprintf("%0.2f", cum)
	case "oz":
		oz := grams * 0.0352739619
		return oz, fmt.Sprintf("%d", int64(oz))
	case "lb":
		lb := grams * 0.0022046226
		return lb, fmt.Sprintf("%.2f", lb)
	case "cuin":
		cuin := NormalLitres * 61.02374409
		return cuin, fmt.Sprintf("%d", int64(cuin))
	case "cuft":
		cuft := NormalLitres * 0.03531466672
		return cuft, fmt.Sprintf("%.2f", cuft)
	case "cuyd":
		cuyd := NormalLitres * 0.001307950619
		return cuyd, fmt.Sprintf("%.3f", cuyd)
	case "aft":
		aft := NormalLitres / 1233481.8375475
		return aft, fmt.Sprintf("%.6f", aft)
	case "tblspus":
		tblsp := NormalLitres * 67.6280454
		return tblsp, fmt.Sprintf("%d", int64(tblsp))
	case "tspus":
		tsp := NormalLitres * 202.8841362
		return tsp, fmt.Sprintf("%d", int64(tsp))
	case "cupus":
		cup := NormalLitres * 4.226752838
		return cup, fmt.Sprintf("%d", int64(cup))
	case "bushelus":
		bushel := NormalLitres * 0.02837759326
		return bushel, fmt.Sprintf("%.2f", bushel)
	case "barrelus":
		barrel := NormalLitres * 0.00628981077
		return barrel, fmt.Sprintf("%.3f", barrel)
	case "galliquid":
		gal := NormalLitres * 0.2641720524
		return gal, fmt.Sprintf("%.1g", gal)
	case "galdry":
		gal := NormalLitres * 2.270207461
		return gal, fmt.Sprintf("%.1g", gal)
	case "flozus":
		oz := NormalLitres * 33.8140227
		return oz, fmt.Sprintf("%d", int64(oz))
	case "ptus":
		pt := NormalLitres * 2.113376419
		return pt, fmt.Sprintf("%d", int64(pt))
	case "qtus":
		qt := NormalLitres * 2.113376419
		return qt, fmt.Sprintf("%d", int64(qt))
	case "tblsp":
		tblsp := NormalLitres * 67.6280454
		return tblsp, fmt.Sprintf("%d", int64(tblsp))
	case "tsp":
		tsp := NormalLitres * 202.8841362
		return tsp, fmt.Sprintf("%d", int64(tsp))
	case "barrel":
		barrel := NormalLitres * 0.006110256897
		return barrel, fmt.Sprintf("%.3f", barrel)
	case "gal":
		gal := NormalLitres * 0.2199692483
		return gal, fmt.Sprintf("%.1f", gal)
	case "floz":
		oz := NormalLitres * 35.19507973
		return oz, fmt.Sprintf("%d", int64(oz))
	case "pt":
		pt := NormalLitres * 1.759753986
		return pt, fmt.Sprintf("%d", int64(pt))
	case "qt":
		qt := NormalLitres * 0.8798769932
		return qt, fmt.Sprintf("%d", int64(qt))
	case "kWhr":
		return kWh, fmt.Sprintf("%.2f", kWh)
	case "kWs":
		kWs := kWh / 3600
		return kWs, fmt.Sprintf("%.4f", kWs)
	case "btu":
		btu := kWh * 3412.141632
		return btu, fmt.Sprintf("%d", int64(btu))
	case "j":
		joules := kWh * 3600000
		return joules, fmt.Sprintf("%d", int64(joules))
	case "cal":
		cal := kWh * 859845.22786
		return cal, fmt.Sprintf("%d", int64(cal))
	case "therm":
		therm := kWh * 0.0341214116
		return therm, fmt.Sprintf("%.2f", therm)
	case "themus":
		therm := kWh * 0.0341295634
		return therm, fmt.Sprintf("%.2f", therm)
	case "hartree":
		ht := kWh * 8.257357615e+23
		return ht, fmt.Sprintf("%d", int64(ht))
	}
	log.Println("Unknown volume type - ", volumeUnits)
	return mols, fmt.Sprintf("%.2f", mols)
}

func getVolumeLabel(units string) string {
	switch units {
	case "kg":
		return "kg"
	case "g":
		return "g"
	case "litres":
		return "L"
	case "cucm":
		return "cm3"
	case "cum":
		return "m3"
	case "oz":
		return "oz"
	case "lb":
		return "lb"
	case "cuin":
		return "cuin"
	case "cuft":
		return "cuft"
	case "cuyd":
		return "cuyd"
	case "aft":
		return "aft"
	case "tblspus":
		return "tblsp"
	case "tspus":
		return "tsp"
	case "cupus":
		return "cups"
	case "bushelus":
		return "bushel"
	case "barrelus":
		return "barrel"
	case "galliquid":
		return "gal"
	case "galdry":
		return "gal"
	case "flozus":
		return "floz"
	case "ptus":
		return "pt"
	case "qtus":
		return "qt"
	case "tblsp":
		return "tblsp"
	case "tsp":
		return "tsp"
	case "barrel":
		return "barrel"
	case "gal":
		return "gal"
	case "floz":
		return "floz"
	case "pt":
		return "pt"
	case "qt":
		return "qt"
	case "kWhr":
		return "kWhr"
	case "kWs":
		return "kWs"
	case "btu":
		return "btu"
	case "j":
		return "joule"
	case "cal":
		return "calorie"
	case "therm":
		return "therm"
	case "thermus":
		return "therm"
	case "hartree":
		return "hartree"
	}
	return "mols"
}
