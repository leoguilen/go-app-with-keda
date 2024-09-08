#!/bin/bash

SQL_SCRIPT="$(dirname $0)/insert_orders.sql"

generate_batch_of_orders () {
   kubectl exec -it deploy/postgres-deployment -- psql -U postgres -d orders -c "$(cat $SQL_SCRIPT)"
}

while :
do
generate_batch_of_orders
sleep 1
done