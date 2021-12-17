# // vim: set ft=bash:
# list all child process for process
list_descendants ()
{
  local children=$(ps -o pid= --ppid "$1")

  for pid in $children
  do
    list_descendants "$pid"
  done

  echo "$children"
}

wait_on_port(){
  local port=$1
  while ! nc -z localhost $port; do
    sleep 0.1 # wait for 1/10 of the second before check again
  done
}

By(){
  echo "$1" >&3
}

force_remove_edgedevice() {
  local device_id=$1
  kubectl get edgedevice $device_id -o=json | jq '.metadata.finalizers = null' | kubectl apply -f -
  kubectl delete edgedevice $1
}

setup() {
  load 'test_helper/bats-support/load'
  load 'test_helper/bats-assert/load'
  By "SETUP: Starting server"
  make run  2>/tmp/server_log_messages &
  echo $! > /tmp/operator-run.pid
  wait_on_port 8888

  By "SETUP: server started"
}


@test 'Register one device' {
  local DEVICE="e5f44aaa-9dfb-408a-9b85-f54c0c3efa02"
  local URL="http://127.0.0.1:8888/api/k4e-management/v1/data/${DEVICE}/out"

  run kubectl get edgedevices --all-namespaces
  assert_equal "${#lines[@]}" "1"

  local data=$(cat <<EOF
{
  "type": "data",
  "message_id": "1",
  "version": 1,
  "directive": "registration",
  "content": {
    "OsImage": "rhel"
  }
}
EOF
)
  run curl \
    -d "${data}" \
    -H 'Content-Type: application/json' \
    $URL
  assert_success

  run kubectl get edgedevices --all-namespaces
  assert_equal "${#lines[@]}" "2"

  run kubectl get edgedevice $DEVICE
  assert_success

  force_remove_edgedevice $DEVICE

  run kubectl get edgedevice $DEVICE
  assert_failure
}

teardown() {
  export PID=$(cat /tmp/operator-run.pid)
  local PROCESS=$(list_descendants $PID)
  By "Killing process: ${PROCESS}"
  kill -9 ${PROCESS}
  rm /tmp/operator-run.pid
}
