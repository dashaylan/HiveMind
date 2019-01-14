./sshpass -p $3 ssh $2@$1 'cd /tmp/hivemind; nohup /tmp/hivemind/hiv >/tmp/hivemind/run.log 2>&1 </dev/null &'
