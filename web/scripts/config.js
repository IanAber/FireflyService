let Electrolysers = [];
let Links = [];

// Remove element at the given index
Array.prototype.remove = function(index) {
    this.splice(index, 1);
}

function setGasDisplayOptions(opt) {
    DisplaySelection = $("#GasDisplayUnits");
    if (opt === null) {
        opt = DisplaySelection.val();
    }
    // DisplaySelection.empty().append($('<option>').val("kWhr").text("Kilowatt Hours"));
    // if ($("#GasUnits").val() === "bar") {
    //     DisplaySelection.append($('<option>').val("bar").text("Bar"));
    //     DisplaySelection.append($('<option>').val("litres").text("Litres"));
    // } else {
    //     DisplaySelection.append($('<option>').val("psi").text("Pounds per Suqare Inch"));
    //     DisplaySelection.append($('<option>').val("cuft").text("Cubic Feet"));
    // }
    DisplaySelection.val(opt);
}

function setupNumericField(inputField, units) {
    let input = document.getElementById(inputField);
    let hiddenValue = document.getElementById(inputField + "Value");
    let unitsValue = document.getElementById(inputField + "Units");
    input.addEventListener("input", () => {
        hiddenValue.innerHTML = input.value;
//        Only show units when there is a value?
        unitsValue.innerHTML = (input.value.length > 0 ? (" " + units) : "");
    });
}

function setupFields() {
    setupNumericField("electrolyserMaxStackVoltsForShutdown", "Volts")
    setupNumericField("electrolyserStopToStartTime", "Minutes")
    setupNumericField("electrolyserStartToStopTime", "Minutes")
    setupNumericField("maxConductivity", "µS/cm")
    setupNumericField("maxGreenConductivity", "µS/cm")
    setupNumericField("maxYellowConductivity", "µS/cm")
    setupNumericField("waterQualityAlarm", "µS/cm")
    setupNumericField("waterDumpSeconds", "Seconds");
    setupNumericField("fcMaxOutput", "kW");
    setupNumericField("fcMaxTime", "Minutes");
    setupNumericField("fcStartSOC", "%");
    setupNumericField("fcStopSOC", "%");
    setupNumericField("GasCapacity", "Nl");
    setupNumericField("GasDetectorThreshold", "");
    setupNumericField("MaxGasPressure", "Bar");
    setupNumericField("GasCapacity", "Nl");
}

function loadSettings() {
    fetch("/getSettings")
        .then( function(response) {
            if (response.status === 200) {
                response.json()
                    .then(function (data) {
                        $("#name").val(data.Name);
                        data.AnalogChannels.forEach(SetAnalogSettings);
                        data.DigitalInputs.forEach(SetDigitalInputSettings);
                        data.DigitalOutputs.forEach(SetDigitalOutputSettings);
                        data.Relays.forEach(SetRelaySettings);
                        data.ACMeasurement.forEach(SetACMeasurementSettings);
                        data.DCMeasurement.forEach(SetDCMeasurementSettings);
                        data.Buttons.forEach(SetButtonName);
                        if (data.FuelCell) {
                            $("#FuelCell").attr('checked', true);
                        }
                        if ((data.electrolyserStopToStartTime >= 1) && (data.electrolyserStopToStartTime <= 20)) {
                            $('#electrolyserStopToStartTime').val(data.electrolyserStopToStartTime);
                        }
                        if ((data.electrolyserStartToStopTime >= 1) && (data.electrolyserStartToStopTime <= 20)) {
                            $('#electrolyserStartToStopTime').val(data.electrolyserStartToStopTime);
                        }
                        if ((data.electrolyserMaxStackVoltsForShutdown >= 25) && (data.electrolyserMaxStackVoltsForShutdown <= 35)) {
                            $("#electrolyserMaxStackVoltsForShutdown").val(data.electrolyserMaxStackVoltsForShutdown);
                        }
                        $("#nodeRed").val(data.nodeRED);
                        $("#subnet").val(data.subnet);
                        $("#APIKey").val(data.apiKey);
                        if (data.FuelCellSettings.IgnoreIsoLow) {
                            $("#isoLowBehaviour").val("true")
                        } else {
                            $("#isoLowBehaviour").val("false")
                        }
                        if (data.FuelCellSettings.Capacity !== "") {
                            $("#fcCapacity").val(data.FuelCellSettings.Capacity);
                        }
                        if (data.FuelCellSettings.Efficiency !== "") {
                            $("#fcEfficiency").val(data.FuelCellSettings.Efficiency);
                        }
                        if (data.FuelCellSettings.StartSOC !== "") {
                            $("#fcStartSOC").val(data.FuelCellSettings.StartSOC);
                        }
                        if (data.FuelCellSettings.StopSOC !== "") {
                            $("#fcStopSOC").val(data.FuelCellSettings.StopSOC);
                        }
                        if (data.FuelCellSettings.MaxRunTime !== "") {
                            $("#fcMaxTime").val(data.FuelCellSettings.MaxRunTime);
                        }
                        if (data.FuelCellSettings.MaximumOutput !== "") {
                            $("#fcMaxOutput").val(data.FuelCellSettings.MaximumOutput);
                        }
                        if (data.links != null) {
                            Links = data.links;
                        }

                        if (data.electrolysers != null) {
                            Electrolysers = data.electrolysers;
                        }
                        let dryerRelay = data.dryerRelay;
                        let dryerRelayControl = $("#dryerRelay");
                        if (dryerRelay === undefined) {
                            dryerRelay = -1;
                        }
                        for (let rl = 0; rl < 16; rl++) {
                            optText = $("#relay"+rl+"name").val();
                            optValue = rl;
                            dryerRelayControl.append($('<option>').val(optValue).text(optText));
                        }
                        dryerRelayControl.val(dryerRelay);
                        let relayNum = data.water;
                        let dumpRelay = $("#waterDumpRelay");
                        for (let rl = 0; rl < 16; rl++) {
                            optText = $("#relay"+rl+"name").val();
                            optValue = rl;
                            dumpRelay.append($('<option>').val(optValue).text(optText));
                        }
                        dumpRelay.val(relayNum);
                        $("#waterDumpSeconds").val(data.waterSeconds);
                        $("#waterDumpAction").val(data.waterDumpAction);
                        $("#maxConductivity").val(data.maxConductivity);
                        $("#waterQualityAlarm").val(data.waterQualityAlarm);
                        RenderElectrolysers();
                        $("#GasDetectorInput").val(data.gasDetectorInput);
                        $("#GasDetectorThreshold").val(data.gasDetectorThreshold);
                        $("#MaxGasPressure").val(data.maxGasPressure);
                        $("#GasCapacity").val(data.gasCapacity);
                        $("#GasVolumeUnits").val(data.gasVolumeUnits);
                        $("#GasLevelType").val(data.gasLevelType);
                        $("#GasPressureUnits").val(data.gasPressureUnits);
                        $("#GasInput").val(data.gasPressureInput);
                        $("#maxYellowConductivity").val(data.conductivityYellowMax);
                        $("#maxGreenConductivity").val(data.conductivityGreenMax);
                        let pumpRelay = $("#coolingPumpRelay");
                        for (let rl = 0; rl < 16; rl++) {
                            optText = $("#relay"+rl+"name").val();
                            optValue = rl;
                            pumpRelay.append($('<option>').val(optValue).text(optText));
                        }
                        pumpRelay.val(data.coolingPumpRelay);
                        $("#coolingPumpStartTemperature").val(data.coolingPumpStartTemperature);
                        $("#coolingPumpStopTemperature").val(data.coolingPumpStopTemperature);
                        $("#boardVersion").text(data.boardVersion);
                        RenderLinks();
                    });
            }
        })
}

function RenderElectrolysers() {
    $("#ElectrolysersBody").empty();
    if (Electrolysers != null) {
        Electrolysers.forEach(function (el) { RenderElectrolyser(el.relay, el.name, el.dryer, el.ip, el.enabled);})
    }
    $('input[type=radio][name=Dryer]').change(function() {
        for (el = 0; el < Electrolysers.length; el++) {
            Electrolysers[el].dryer = (el === this.value);
        }
    });
}

function RenderElectrolyser(relayNum, relayName, Dryer, ip, enabled) {
    let numElectrolysers = $("#ElectrolysersBody tr").length;
    let selected = '';
    if (relayNum < 0) {
        selected = ' selected';
    }
    let selectOptions = '<option value="-1"' + selected + '>Select a Relay</option>';
    let nameID = "el" + numElectrolysers + "Name";
    let relayID = "el" + numElectrolysers + "Relay";
    let ipID = "el" + numElectrolysers + "IP";
//    let dryerID = "Dryer" + numElectrolysers;
    let elEnabledID = "el" + numElectrolysers + "Enabled";
    for (let rl = 0; rl < 16; rl++) {
        if (relayNum === rl) {
            selected = ' selected';
        } else {
            selected = '';
        }
        let option = '<option value="' + rl + '"' + selected + ' >' + $("#relay" + rl + "name").val() + '</option>';
        selectOptions += option;
    }
    let IsEnabled = enabled ? "checked" : "";
    let newRow = '<tr class="elSetting" id="el' + numElectrolysers + 'Row">';
    newRow += '<td class="elRelaySetting"><select id="' + relayID + '" name="' + relayID + '">' + selectOptions + '</select></td>';
    newRow += '<td class="elNameSetting"><input class="settings" type="text" id="' + nameID + '" name="' + nameID + '" value="' + relayName + '"></td>';
//    newRow += '<td class="elDryerSetting"><input class="settings_cb" type="radio" id="' + dryerID + '" name="Dryer" value=' + numElectrolysers + '><label for="' + dryerID + '">Dryer Control</label></td>';
    newRow += '<td class="elIP"><span class="settings" id="' + ipID + '">' + ip + '</span></td>';
    newRow += '<td class="elEnabled"><input class="settings" type="checkbox" id="' + elEnabledID + '" name="' + elEnabledID + '" value="Enabled" ' + IsEnabled + '></td>'
    newRow += '<td class="elDelete"><img src="images/trash.png" alt="Delete" onclick="deleteElectrolyser(' + numElectrolysers + ')" class="button" /></td></tr>';
    $("#ElectrolysersBody").append(newRow);
    $('#'+relayID).on("change", function() {
        Electrolysers[numElectrolysers].relay = parseInt($(this).val(), 10);
    });
    $('#'+nameID).on("change", function() {
        Electrolysers[numElectrolysers].name = $(this).val();
    });
    $('#'+elEnabledID).on("click", function() {
        Electrolysers[numElectrolysers].enabled = $(this).prop("checked");
    })
}

function deleteElectrolyser(num) {
    let isDryer = Electrolysers[num].Dryer;
    Electrolysers.remove(num);
    RenderElectrolysers();
    if (isDryer) {
        $("#DryerNone").prop("checked", true);
    }
}

function appendElectrolyser() {
    Electrolysers.push({relay:-1, name:"", dryer:false, enabled:false});
    RenderElectrolysers();
}

function validateElectrolyserRelays() {
    for (el = 0; el < Electrolysers.length; el++) {
        if (Electrolysers[el].relay < 0) {
            alert("Please select a valid relay for all defined electrolysers.");
            return false;
        }
        for (elComp = el + 1; elComp < Electrolysers.length; elComp++) {
            if (Electrolysers[el].relay === Electrolysers[elComp].relay) {
                alert("All electrolyser relays must be unique!");
                return false;
            }
        }
        for (elComp = el + 1; elComp < Electrolysers.length; elComp++) {
            if (Electrolysers[el].name.toLowerCase() === Electrolysers[elComp].name.toLowerCase()) {
                alert("All electrolyser names must be unique!");
                return false;
            }
        }

    }
    return true;
}

function GenerateNewKey() {
    $.ajax({
        url: '/generateKey',
        type: 'get',
        headers: {
            "Content-Type": "application/json"
        },
        dataType: 'json',
        success: function(response) {
            if (response.guid !== "") {
                $("#APIKey").val(response.guid);
            }
        },
        error: function (xhr, ajaxOptions, thrownError) {
            if (xhr.status === 400) {
                alert(xhr.responseJSON.errors[0].Err);
            } else {
                alert(xhr.status + " : " + thrownError);
            }
        }
    });

}

function SetAnalogSettings(channel){
    $("#a"+channel.Port+"name").val(channel.Name);
    $("#a"+channel.Port+"LowVal").val(channel.LowerCalibrationActual);
    $("#a"+channel.Port+"LowA2D").val(channel.LowerCalibrationAtoD);
    $("#a"+channel.Port+"HighVal").val(channel.UpperCalibrationActual);
    $("#a"+channel.Port+"HighA2D").val(channel.UpperCalibrationAtoD);
    $("#a"+channel.Port+"MaxVal").val(channel.MaxVal);
    $("#a"+channel.Port+"MinVal").val(channel.MinVal);

    // Add options for gas input and detector
    $("#GasInput").append(new Option(channel.Name, channel.Port));
    $("#GasDetectorInput").append((new Option(channel.Name, channel.Port)));
}

function SetButtonName(button, idx) {
    $("#btn"+idx+"name").val(button.Name);
    $("#btn"+idx+"user").prop('checked',button.ShowOnCustomer);
}

function SetRelaySettings(channel) {
    $("#relay"+channel.Port+"name").val(channel.Name);
}

function SetDigitalOutputSettings(channel) {
    $("#do"+channel.Port+"name").val(channel.Name);
}

function SetDigitalInputSettings(channel) {
    $("#di"+channel.Port+"name").val(channel.Name);
    $("#di"+channel.Port+"user").prop('checked',channel.ShowOnCustomer);
}

function SetACMeasurementSettings(channel) {
    $("#ACMeasurement"+channel.SlaveID).val(channel.Name);
}

function SetDCMeasurementSettings(channel) {
    $("#DCMeasurement"+channel.SlaveID).val(channel.Name);
    if (channel.Name !== "") {
        $("#calibrateDC" + channel.SlaveID).show();
    } else {
        $("#calibrateDC" + channel.SlaveID).hide();
    }
}

function saveSettings() {
    if (validateElectrolyserRelays()) {
        let relayJSON = JSON.stringify(Electrolysers);
        $("#ElectrolyserRelays").val(relayJSON);
        let linkJSON = JSON.stringify(Links);
        $("#Links").val(linkJSON);
        $("#settingsForm").submit();
    }
}

function RenderLinks() {
    $("#LinksBody").empty();
    if (Links != null) {
        Links.forEach(function (lk) { RenderLink(lk); } );
    }
}

function RenderLink(lk) {
    let numLinks = $("#LinksBody tr").length;
    let nameID = "lk" + numLinks + "Name";
    let internalID = "lk" + numLinks + "internal";
    let externalID = "lk" + numLinks + "external";
    let showOnCustomerScreen = "lk" + numLinks + "customer";

    let IsCustomerEnabled = lk.showOnCustomerScreen ? "checked" : "";
    let newRow = '<tr class="lkSetting" id="lk' + numLinks + 'Row">';
    newRow += '<td class="lkSetting"><input class="settings" type="text" id="' + nameID + '" name="' + nameID + '" value="' + lk.name + '"></td>';
    newRow += '<td class="lkSetting"><input class="settings" type="text" id="' + internalID + '" name="' + internalID + '" value="' + lk.internal + '"></td>';
    newRow += '<td class="lkSetting"><input class="settings" type="text" id="' + externalID + '" name="' + externalID + '" value="' + lk.external + '"></td>';
    newRow += '<td class="lkShowOnCustomer"><input class="settings" type="checkbox" id="' + showOnCustomerScreen + '" name="' + showOnCustomerScreen + '" value="Enabled" ' + IsCustomerEnabled + '></td>'
    newRow += '<td><img src="images/trash.png" alt="Delete" onclick="deleteLink(' + numLinks + ')" class="button" /></td></tr>';

    $("#LinksBody").append(newRow);

    $('#'+nameID).on("change", function() {
        Links[numLinks].name = $(this).val();
    });
    $('#'+internalID).on("change", function() {
        Links[numLinks].internal = $(this).val();
    });
    $('#'+externalID).on("change", function() {
        Links[numLinks].external = $(this).val();
    });
    $('#'+showOnCustomerScreen).on("change", function() {
        Links[numLinks].showOnCustomerScreen = $(this).prop("checked");
    })
}

function deleteLink(num) {
    Links.remove(num);
    RenderLinks();
}

function appendLink() {
    Links.push({name:"", internal:"", external:"", showOnCustomerScreen:false});
    RenderLinks();
}
