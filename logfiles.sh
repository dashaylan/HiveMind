#!/bin/bash

#Grab all the logs


sshpass -p "DroneDrone01" scp -r drone1@20.36.20.142:/home/drone1/8D/Pi0-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone02" scp -r drone2@20.190.48.49:/tmp/hivemind/Pi1-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone03" scp -r drone3@52.151.56.206:/tmp/hivemind/Pi2-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone04" scp -r drone4@52.229.50.134:/tmp/hivemind/Pi3-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone05" scp -r drone5@52.229.58.24:/tmp/hivemind/Pi4-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone06" scp -r drone6@40.65.105.92:/tmp/hivemind/Pi5-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone07" scp -r drone7@40.65.111.58:/tmp/hivemind/Pi6-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone08" scp -r drone8@52.151.41.54:/tmp/hivemind/Pi7-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
