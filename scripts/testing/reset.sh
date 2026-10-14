#!/bin/bash
set -e
docker exec -i -e MYSQL_PWD=root ttpg-mysql mysql -uroot -h127.0.0.1 ttpg -e "TRUNCATE terminal_boardingpass; TRUNCATE tbl_tujuan; TRUNCATE tbl_po;" 2>/dev/null
docker exec -i ttpg-mysql mysql -uroot -proot -h127.0.0.1 --force ttpg < $(dirname "$0")/ttpg_schema.sql 2>/dev/null || true
docker exec -i ttpg-mysql mysql -uroot -proot -h127.0.0.1 --force < $(dirname "$0")/../../migrations/mysql/004_create_ttpg_users.sql 2>/dev/null
docker exec -i -e MYSQL_PWD=root ttpg-mysql mysql -uroot -h127.0.0.1 -e "GRANT ALL PRIVILEGES ON ttpg_users.* TO 'api2'@'%'" 2>/dev/null
docker exec -i -e MYSQL_PWD=root ttpg-mysql mysql -uroot -h127.0.0.1 ttpg_users -e "UPDATE user SET po_id='' WHERE username='admin'" 2>/dev/null
docker exec -i -e MYSQL_PWD=root ttpg-mysql mysql -uroot -h127.0.0.1 ttpg_penginapan < $(dirname "$0")/../../migrations/mysql/002_create_ttpg_penginapan.sql 2>/dev/null
docker exec -i ttpg-mongo mongosh --quiet ttpg --eval '
  db.boarding.deleteMany({ticket_id:/^(T001|MB-)/});
  db.boardingpass.deleteMany({ticket_id:"T001"});
  db.boarding.insertOne({ticket_id:"T001", date_of_departure:new Date().toISOString().slice(0,10), time_of_departure:"08:30:00",
    nopol:"B 9999 XY", po_id:"PO001", destination_id:2, destination:{tujuan:"Bekasi"},
    po:{nama_po:"PO Contoh Utama", nama_singkat:"PO001"},
    pnp:[{name:"Dewi Lestari", gender:"P"}], user_id:1, user:{username:"loket"},
    status:"active", createdAt:new Date(), updatedAt:new Date()});' > /dev/null
docker exec -i -e MYSQL_PWD=root ttpg-mysql mysql -uroot -h127.0.0.1 ttpg -e "INSERT INTO terminal_boardingpass (noreg,ticket_id,id_po,id_tujuan,nopol,tanggal_berangkat,jam_berangkat,nama_pnp,sex_pnp,fungsi,gate_id,device_id,time_tab,retribusi,count_scan,pembelian_dari,created,updated) VALUES ('GATE002','T002','PO001',2,'B 5555 XY',CURDATE(),'12:00:00','Uji Coba','L','gatein',1,'10.0.0.2',NOW(),0,1,'loket',NOW(),NOW())" 2>/dev/null
echo "RESET OK"
