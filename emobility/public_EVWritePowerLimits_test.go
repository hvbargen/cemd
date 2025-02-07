package emobility

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/enbility/eebus-go/spine/model"
	"github.com/enbility/eebus-go/util"
	"github.com/stretchr/testify/assert"
)

func Test_EVWritePowerLimits(t *testing.T) {
	emobilty, eebusService := setupEmobility()

	data := []EVDurationSlotValue{}

	err := emobilty.EVWritePowerLimits(data)
	assert.NotNil(t, err)

	localDevice, remoteDevice, entites, writeHandler := setupDevices(eebusService)
	emobilty.evseEntity = entites[0]
	emobilty.evEntity = entites[1]

	err = emobilty.EVWritePowerLimits(data)
	assert.NotNil(t, err)

	emobilty.evTimeSeries = timeSeriesConfiguration(localDevice, emobilty.evEntity)

	err = emobilty.EVWritePowerLimits(data)
	assert.NotNil(t, err)

	datagram := datagramForEntityAndFeatures(false, localDevice, emobilty.evEntity, model.FeatureTypeTypeTimeSeries, model.RoleTypeServer, model.RoleTypeClient)

	cmd := []model.CmdType{{
		TimeSeriesDescriptionListData: &model.TimeSeriesDescriptionListDataType{
			TimeSeriesDescriptionData: []model.TimeSeriesDescriptionDataType{
				{
					TimeSeriesId:   util.Ptr(model.TimeSeriesIdType(0)),
					TimeSeriesType: util.Ptr(model.TimeSeriesTypeTypeConstraints),
				},
			},
		}}}
	datagram.Payload.Cmd = cmd

	err = localDevice.ProcessCmd(datagram, remoteDevice)
	assert.Nil(t, err)

	err = emobilty.EVWritePowerLimits(data)
	assert.NotNil(t, err)

	type dataStruct struct {
		error              bool
		minSlots, maxSlots uint
		slots              []EVDurationSlotValue
	}

	tests := []struct {
		name string
		data []dataStruct
	}{
		{
			"too few slots",
			[]dataStruct{
				{
					true, 2, 2,
					[]EVDurationSlotValue{
						{Duration: time.Hour, Value: 11000},
					},
				},
			},
		}, {
			"too many slots",
			[]dataStruct{
				{
					true, 1, 1,
					[]EVDurationSlotValue{
						{Duration: time.Hour, Value: 11000},
						{Duration: time.Hour, Value: 11000},
					},
				},
			},
		},
		{
			"1 slot",
			[]dataStruct{
				{
					false, 1, 1,
					[]EVDurationSlotValue{
						{Duration: time.Hour, Value: 11000},
					},
				},
			},
		},
		{
			"2 slots",
			[]dataStruct{
				{
					false, 1, 2,
					[]EVDurationSlotValue{
						{Duration: time.Hour, Value: 11000},
						{Duration: 30 * time.Minute, Value: 5000},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, data := range tc.data {
				datagram = datagramForEntityAndFeatures(false, localDevice, emobilty.evEntity, model.FeatureTypeTypeTimeSeries, model.RoleTypeServer, model.RoleTypeClient)

				cmd = []model.CmdType{{
					TimeSeriesConstraintsListData: &model.TimeSeriesConstraintsListDataType{
						TimeSeriesConstraintsData: []model.TimeSeriesConstraintsDataType{
							{
								TimeSeriesId: util.Ptr(model.TimeSeriesIdType(0)),
								SlotCountMin: util.Ptr(model.TimeSeriesSlotCountType(data.minSlots)),
								SlotCountMax: util.Ptr(model.TimeSeriesSlotCountType(data.maxSlots)),
							},
						},
					}}}
				datagram.Payload.Cmd = cmd

				err = localDevice.ProcessCmd(datagram, remoteDevice)
				assert.Nil(t, err)

				err = emobilty.EVWritePowerLimits(data.slots)
				if data.error {
					assert.NotNil(t, err)
					continue
				} else {
					assert.Nil(t, err)
				}

				sentDatagram := model.Datagram{}
				sentBytes := writeHandler.LastMessage()
				err := json.Unmarshal(sentBytes, &sentDatagram)
				assert.Nil(t, err)

				sentCmd := sentDatagram.Datagram.Payload.Cmd
				assert.Equal(t, 1, len(sentCmd))

				sentPowerLimitsData := sentCmd[0].TimeSeriesListData.TimeSeriesData[0].TimeSeriesSlot
				assert.Equal(t, len(data.slots), len(sentPowerLimitsData))

				for index, item := range sentPowerLimitsData {
					assert.Equal(t, data.slots[index].Value, item.MaxValue.GetValue())
				}
			}
		})
	}
}
