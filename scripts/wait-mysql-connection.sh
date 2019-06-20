#!bin/sh

until nc -z -v -w30 mysql 3306
do
  echo "Waiting for database connection..."
  # wait for 3 seconds before check again
  sleep 3
done

/migrate -path=/migrations/ -database "mysql://root:mysql@tcp(mysql:3306)/app?charset=utf8mb4" up
