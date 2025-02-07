package emobility

import (
	"encoding/json"
	"testing"
	"time"

	"github.com/enbility/eebus-go/spine/model"
	"github.com/enbility/eebus-go/util"
	"github.com/stretchr/testify/assert"
)

func Test_EVWriteIncentives(t *testing.T) {
	emobilty, eebusService := setupEmobility()

	data := []EVDurationSlotValue{}

	err := emobilty.EVWriteIncentives(data)
	assert.NotNil(t, err)

	localDevice, remoteDevice, entites, writeHandler := setupDevices(eebusService)
	emobilty.evseEntity = entites[0]
	emobilty.evEntity = entites[1]

	err = emobilty.EVWriteIncentives(data)
	assert.NotNil(t, err)

	emobilty.evIncentiveTable = incentiveTableConfiguration(localDevice, emobilty.evEntity)

	err = emobilty.EVWriteIncentives(data)
	assert.NotNil(t, err)

	datagram := datagramForEntityAndFeatures(false, localDevice, emobilty.evEntity, model.FeatureTypeTypeIncentiveTable, model.RoleTypeServer, model.RoleTypeClient)

	cmd := []model.CmdType{{
		IncentiveTableConstraintsData: &model.IncentiveTableConstraintsDataType{
			IncentiveTableConstraints: []model.IncentiveTableConstraintsType{
				{
					IncentiveSlotConstraints: &model.TimeTableConstraintsDataType{
						SlotCountMin: util.Ptr(model.TimeSlotCountType(1)),
						SlotCountMax: util.Ptr(model.TimeSlotCountType(10)),
					},
				},
			},
		}}}
	datagram.Payload.Cmd = cmd

	err = localDevice.ProcessCmd(datagram, remoteDevice)
	assert.Nil(t, err)

	err = emobilty.EVWriteIncentives(data)
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
						{Duration: time.Hour, Value: 0.1},
					},
				},
			},
		}, {
			"too many slots",
			[]dataStruct{
				{
					true, 1, 1,
					[]EVDurationSlotValue{
						{Duration: time.Hour, Value: 0.1},
						{Duration: time.Hour, Value: 0.1},
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
						{Duration: time.Hour, Value: 0.1},
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
						{Duration: time.Hour, Value: 0.1},
						{Duration: 30 * time.Minute, Value: 0.2},
					},
				},
			},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			for _, data := range tc.data {
				datagram = datagramForEntityAndFeatures(false, localDevice, emobilty.evEntity, model.FeatureTypeTypeIncentiveTable, model.RoleTypeServer, model.RoleTypeClient)

				cmd = []model.CmdType{{
					IncentiveTableConstraintsData: &model.IncentiveTableConstraintsDataType{
						IncentiveTableConstraints: []model.IncentiveTableConstraintsType{
							{
								IncentiveSlotConstraints: &model.TimeTableConstraintsDataType{
									SlotCountMin: util.Ptr(model.TimeSlotCountType(data.minSlots)),
									SlotCountMax: util.Ptr(model.TimeSlotCountType(data.maxSlots)),
								},
							},
						},
					}}}
				datagram.Payload.Cmd = cmd

				err = localDevice.ProcessCmd(datagram, remoteDevice)
				assert.Nil(t, err)

				err = emobilty.EVWriteIncentives(data.slots)
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

				sentIncentiveData := sentCmd[0].IncentiveTableData.IncentiveTable[0].IncentiveSlot
				assert.Equal(t, len(data.slots), len(sentIncentiveData))

				for index, item := range sentIncentiveData {
					assert.Equal(t, data.slots[index].Value, item.Tier[0].Incentive[0].Value.GetValue())
				}
			}
		})
	}
}
