Полная документация проекта Robbo Personal Account Backend
1. Структура проекта
robbo_personal_account_backend/
├── app/                          # Конфигурация приложения (DI)
├── graph/                        # GraphQL схемы и сгенерированные резолверы
│   ├── *.graphqls               # Схемы (auth, user, student, teacher, etc.)
│   └── generated/               # Сгенерированный код
├── package/
│   ├── auth/                    # Аутентификация (JWT, токены)
│   ├── users/                   # Управление пользователями
│   ├── courses/                 # Управление курсами
│   ├── projects/                # Проекты Scratch
│   ├── projectPage/             # Страницы проектов
│   ├── robboUnits/              # Подразделения (филиалы)
│   ├── robboGroup/              # Группы (классы)
│   ├── coursePacket/            # Пакеты курсов
│   ├── cohorts/                 # Когорты
│   ├── config/                  # Конфигурация
│   ├── db_client/               # PostgreSQL клиент
│   ├── edx/                    # Интеграция с EdX LMS
│   ├── logger/                  # Логирование
│   ├── models/                  # Модели базы данных
│   └── resolvers/               # GraphQL резолверы
├── server/                      # HTTP сервер и роутинг
├── frontend/                    # HTML страницы
├── tests/                       # Интеграционные тесты
├── main.go                     # Точка входа
├── docker-compose.yml          # Docker оркестрация
├── Dockerfile                   # Docker образ
└── go.mod / go.sum             # Зависимости Go

3. GraphQL API
3.1 Файлы схем (graph/*.graphqls)
Файл
auth.graphqls
user.graphqls
student.graphqls
teacher.graphqls
parent.graphqls
unitadmin.graphqls
robboUnit.graphqls
robboGroup.graphqls
projectPage.graphqls
courses.graphqls
error.graphqls
Playground: http://localhost:8080/
Endpoint: http://localhost:8080/query
---
4. Модели базы данных
Файлы: package/models/*.go
Модель
user.go
student.go
teacher.go
parent.go
unitadmin.go
superadmin.go
freeListener.go
robboUnit.go
robboGroup.go
courses.go
cohorts.go
coursePacket.go
projects.go
projectsPage.go
courseRelation.go
media.go, image.go, absolutemedia.go, courseapimediacollection.go
childrenOfParent.go, studentsOfTeacher.go, teacher'sRobboGroups.go, unitadmin'sRobboUnits.go
---
5. Аутентификация
5.1 Flow
1. Sign Up: POST /auth/sign-up
   → Создает пользователя в БД
   → Хеширует пароль (SHA1 + salt)
   → Возвращает access + refresh токены
2. Sign In: POST /auth/sign-in  
   → Проверяет email/password
   → Возвращает JWT токены
   → Refresh токен в HTTP-only cookie
3. Refresh: GET /auth/refresh
   → Использует refresh токен из cookie
   → Выдает новый access токен
4. API вызовы:
   → Header: Authorization: Bearer <access_token>
   → Middleware проверяет токен
   → Извлекает user_id и role в контекст
5.2 Конфигурация (package/config/config.yml)
auth:
  access_signing_key: "access_signing_key"
  refresh_signing_key: "refresh_signing_key"
  access_token_ttl: 300        # 5 минут
  refresh_token_ttl: 604800    # 7 дней
5.3 Middleware
Файл: server/middleware.go
// Извлекает Bearer токен
// Парсит JWT
// Устанавливает user_id и user_role в контекст Gin
---
6. Роли пользователей
Файл: package/models/user.go
Роль
Student
Teacher
Parent
FreeListener
UnitAdmin
SuperAdmin
Anonymous
WithExpiredToken
Права доступа
Сущность
Student CRUD
Teacher CRUD
UnitAdmin CRUD
RobboUnit
RobboGroup
Course Create/Update
Course View
Projects
ProjectPages
---
7. Интеграция с EdX
Файл: package/edx/
Система интегрируется с внешней EdX LMS:
api_urls:
  refreshToken: "https://edx-test.ru/oauth2/access_token"
  getUser: "https://edx-test.ru/api/user/v1/me"
  getCourse: "https://edx-test.ru/api/courses/v1/courses/"
  getEnrollment: "https://edx-test.ru/api/enrollment/v1/enrollments?username="
  getCourses: "https://edx-test.ru/api/courses/v1/courses/"
  postEnrollment: "https://edx-test.ru/api/enrollment/v1/enrollment"
---