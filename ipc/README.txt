HOW TO RUN TEST PROGRAM ON AZURE

1. ssh into drone1:

Username: drone1 Password: dronedrone01

[COMMAND:]
ssh drone1@20.36.20.142

-----------------------------------------------------

2. ssh into drone2:

This is just to make sure drone2 is alive

Username: drone2 Password: dronedrone02

[COMMAND:]
ssh drone2@20.190.48.49

-----------------------------------------------------

3. Kill any ongoing hiv process on drone2

This will remove any process that is using our :6464 port.

[COMMAND:]
fuser 6464/tcp
kill -s 9 <process id>

-----------------------------------------------------

4. If not touched, code should be already there on drone 1's proj2 directory.

The pi_hivemind.go is the attempt for a remote version of pi.go.
The simple_program_remote is a remote version of simpledsm.go.

[COMMAND:]
go run pi_hivemind.go

[COMMAND:]
go run simple_program_remote.go

[NOTE:]
Make sure the config.json has the following information (for running with 2 drones only)
{
    "IsCBM": true,
    "Drones": [
        {
            "Address": "20.190.48.49",
            "Port": "",
            "Username": "drone2",
            "Password": "DroneDrone02"
        }
    ],
    "CBM": "20.36.20.142",
    "DroneList": null
}

-----------------------------------------------------

5. On drone2, check the log

[COMMAND:]
cd /tmp/hivemind
cat run.log