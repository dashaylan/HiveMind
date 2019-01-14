#!/bin/bash

#Grab all the logs TIPC run


sshpass -p "DroneDrone01" scp -r drone1@20.36.20.142:/home/drone1/Test/Pi0-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone02" scp -r drone2@20.190.48.49:/home/drone2/Test/Pi1-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone03" scp -r drone3@52.151.56.206:/home/drone3/Test/Pi2-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone04" scp -r drone4@52.229.50.134:/home/drone4/Test/Pi3-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone05" scp -r drone5@52.229.58.24:/home/drone5/Test/Pi4-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone06" scp -r drone6@40.65.105.92:/home/drone6/Test/Pi5-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone07" scp -r drone7@40.65.111.58:/home/drone7/Test/Pi6-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
sshpass -p "DroneDrone08" scp -r drone8@52.151.41.54:/home/drone8/Test/Pi7-Log.txt /home/dashaylan/Documents/CPSC416/P2/Logs
