let Electrolysers = [];
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
                        $("#electrolyserHoldOffTime").val(data.electrolyserHoldOffTime / 1000000000);
                        $("#electrolyserHoldOnTime").val(data.electrolyserHoldOnTime / 1000000000);
                        $("#electrolyserOffDelay").val(data.electrolyserOffDelay / 1000000000);
                        $("#electrolyserShutDownDelay").val(data.electrolyserShutDownDelay / 1000000000);
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
                        if (data.electrolysers != null) {
                            Electrolysers = data.electrolysers;
                        }
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
                        $("#GasUnits").val(data.gasUnits);
                        setGasDisplayOptions(data.gasDisplayUnits);
                        $("#GasDisplayUnits").val(data.gasDisplayUnits);
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
    let dryerID = "Dryer" + numElectrolysers;
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
    newRow += '<td class="elDryerSetting"><input class="settings_cb" type="radio" id="' + dryerID + '" name="Dryer" value=' + numElectrolysers + '><label for="' + dryerID + '">Dryer Control</label></td>';
    newRow += '<td class="elIP"><span class="settings" id="' + ipID + '">' + ip + '</span></td>';
    newRow += '<td class="elEnabled"><input class="settings" type="checkbox" id="' + elEnabledID + '" name="' + elEnabledID + '" value="Enabled" ' + IsEnabled + '></td>'
    newRow += '<td><img src="images/trash.png" alt="Delete" onclick="deleteElectrolyser(' + numElectrolysers + ')" class="button" /></td></tr>';
    $("#ElectrolysersBody").append(newRow);
    if (Dryer) {
        $("#Dryer"+numElectrolysers).prop("checked", true);
    }
    $('#'+relayID).on("change", function() {
        Electrolysers[numElectrolysers].relay = parseInt($(this).val(), 10);
    });
    $('#'+nameID).on("change", function() {
        Electrolysers[numElectrolysers].name = $(this).val();
    });
    $('#'+elEnabledID).on("change", function() {
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
        let relayJSON = JSON.stringify(Electrolysers)
        $("#ElectrolyserRelays").val(relayJSON);
        $("#settingsForm").submit();
    }
}