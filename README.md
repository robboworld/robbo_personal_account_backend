# robbo_student_personal_account

Licensed under the Apache License, Version 2.0. See [LICENSE](LICENSE) and [NOTICE](NOTICE).

### dev cmd
### generate GraphQL files
go run github.com/99designs/gqlgen generate --verbose

### cloning repository
git clone https://github.com/skinnykaen/robbo_student_personal_account.git
### open folder
cd robbo_student_personal_account
### download dependency and libraries 
go mod download
### setup postgres
docker-compose up -d
### run app
cd cmd
go run main.go