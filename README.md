# Contest Management System

### Система для группировки контестов и управления ими

> Группа - объединение контестов, редактированием которой занимается команда администраторов \
> Контест - набор задач, ссылка на тестирующую систему и условия к задачам \
> Хост - право администрирования конкретной группы

## Описание роутов

### Get-эндпоинты
    
* / - редиректит на /api/users/groups
* /api/users/groups - список всех зарегестрированных групп. Первое, что видит пользователь, заходящий на страницу. Возвращает json, содержащий список номеров [3](#group-json)
* /api/admins/home - домашняя страница админов (доступна только после логина)
* /api/users/groups/x - список контестов в группе с данным id=x. Доступен всем. Возвращает json номер [5](#all-contests-json)
* /api/inner/contest/x - информация о контесте с id=x. Доступен всем. Возвращает json номер [2](#contest-json)

### Post-эндпоинты
    
* /api/shutdown - выключает приложение. Доступен для выполнения только от лица первого [модератора](#система-модераторов)
* /api/inner/register_admin - используется, чтобы добавить в базу данных логин-пароль для нового админа. Выполняется от лица администратора, \
 принимает json номер [1](#admin-json)
* /api/admins/login - войти в систему. Принимает json номер [1](#admin-json). Создает на сервере куки с вашей сессией, которая живет количество часов, указанное в конфиге
* /api/admins/create_contest - создает новый контест в группе, название которой написано в хедере с ключом Group. Также требует json номер [2](#contest-json). \
Вы должны быть хостом в данной группе.
* /api/admins/create_group - создает новую группу. Вы должны быть модератором, чтобы сделать это. В новой группе вам и 1-ому модератору автоматически будет выдан статус хоста. \
Принимает json номер [3](#group-json)
* /api/admins/logout - выйти из текущего пользователя. Удаляет куки с вашей сессией и ее саму с сервера.
* /api/admins/give_host - выдает модератору хоста в группе с указанным номером. Вы должны быть хостом в данной группе. Принимает json номер [4](#group-host-json).
* /api/admins/edit_contest - изменяет контест, название которого передано в хедерах с ключом Contest. Также принимает json номер [2](#contest-json), в котором описано, \
как должен выглядеть контест после изменения (все поля!). Вы должны быть хостом в группе, где находится контест.
* /api/admins/remove_host - забирает хоста в заданной группе. Принимает json номер [4](#group-host-json). Вы должны быть хостом в данной группе, при этом нельзя \
забрать хоста у модератора с id=1 (дефолтного).

## Система модераторов

В [конфиге](#конфиг), среди прочего, можно (и нужно!) указать логин-пароль модератора, который будет всегда иметь id=1. Он же дефолтный, имеющий хоста во всех группах навсегда. \
Чтобы создать нового модератора, надо придумать ему логин-пароль и использовать соответствующий эндпоинт. "Новорожденный", если не дать ему нигде хоста, сможет только создать свою \
группу и там делать что угодно. \
Первый модератор автоматически имеет хоста во всех группах и снять с него эту должность нельзя. Все остальные модераторы и хосты равноправны между собой.

## JSON

### Admin json
Содержит 2 ключа: "login" и "password", оба строковые.

### Contest json
Содержит 5 ключей: "name" - название контеста, "url" - ссылка на тестирующую систему, "contestPicture" - пока что не используется, ибо mvp, "comment" - описание контеста, \
"statementsUrl" - ссылка на условия задач. Все строковые.

### Group json
Содержит 3 ключа: "name" - название группы, "groupPicture" - пока что не используется, ибо mvp, "description" - описание группы. Все строковые.

### Group host json
Содержит 2 ключа: "moderatorId" - логин модератора, которому выдается хост, и "groupId" - номер группы, в которой выдается хост.

### All contests json
Содержит 1 ключ: "contests" - список id контестов в данной группе

## Конфиг

Конфиг лежит в config.yaml файле, который должен находится в корне проекта, на одном уровне с go.mod
- db_path - путь к файлу с базой данных относительно корня проекта. Т.е. если вы хотите, чтобы полный путь выглядел как "/user/projects/cms/resources/main.db", то значение \
этого ключа должно быть "./resources/main.db"
- log_path - путь к файлу с логами относительно корня проекта, так же, как в предыдущем ключе.
- port - порт, который будешь слушать сервер. В формате ":5757"
- is_debug - булевый параметр, меняющий некоторые вещи для удобства отладки (например, длина сессии устанвливается в одну минуту)
- admin_login - логин дефолтного админа
- admin_password - пароль дефолтного админа
- session_expiry_time - длина сессии, в часах. Работает только при is_debug = false, не принимает значения <1
- cookie_expiry_time - длина жизни создаваемой куки, в часах. Работает только при is_debug = false, не принимает значения <1

## Настройка и запуск
### TODO