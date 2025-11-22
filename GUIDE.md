# Руководство по проекту

### Назначение ревьюверов

При создании PR автоматически назначается до 2 случайных активных ревьюверов из команды автора:

- Исключается автор PR из кандидатов
- Если активных пользователей меньше 2, назначается столько, сколько доступно (0-2)

### Переназначение ревьювера

- Новый ревьювер выбирается из команды старого ревьювера (не автора PR)
- Требуется наличие хотя бы одного активного кандидата в команде
- Проверяется, что старый ревьювер действительно был назначен
- Запрещено для PR со статусом MERGED

### Идемпотентность merge

Повторный вызов merge не меняет `merged_at`, если он уже был установлен. Реализовал через SQL:

```sql
UPDATE pull_requests
SET status = 'MERGED', merged_at = COALESCE(merged_at, NOW())
WHERE pull_request_id = $1
```

### Обработка ошибок

Формат ошибок согласно OpenAPI спецификации:

- `400` - TEAM_EXISTS, PR_EXISTS, invalid request body
- `404` - NOT_FOUND (команда, пользователь, PR не найдены)
- `409` - PR_MERGED, NOT_ASSIGNED, NO_CANDIDATE

## Примеры использования API

### Полный сценарий работы

```bash
# 1. Создание team
curl -X POST http://localhost:8080/team/add \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "members": [
      {"user_id": "u1", "username": "Andrey Arshavin", "is_active": true},
      {"user_id": "u2", "username": "Yury Zhirkov", "is_active": true},
      {"user_id": "u3", "username": "Roman Pavluchenko", "is_active": true}
    ]
  }'

# 2. Создание PR
curl -X POST http://localhost:8080/pullRequest/create \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1337",
    "pull_request_name": "Add search",
    "author_id": "u1"
  }'

# 3. Получение PR'ы ревьювера
curl "http://localhost:8080/users/getReview?user_id=u2"

# 4. Переназначение ревьювера
curl -X POST http://localhost:8080/pullRequest/reassign \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1337",
    "old_user_id": "u2"
  }'

# 5. Получить статистику
curl http://localhost:8080/stats

# 6. Деактивирование пользователей
curl -X POST http://localhost:8080/users/deactivate \
  -H "Content-Type: application/json" \
  -d '{
    "team_name": "backend",
    "user_ids": ["u2", "u3"]
  }'

# 7. Merge PR
curl -X POST http://localhost:8080/pullRequest/merge \
  -H "Content-Type: application/json" \
  -d '{
    "pull_request_id": "pr-1001"
  }'
```

## Проблемы, с которыми столкнулся

### 1. Массив параметров PostgreSQL

**Проблема**: Изначально пытался передать `[]string` напрямую в SQL запрос при деактивации пользователей, но с PostgreSQL это невозможно.

**Решение**: Через `pq.Array()`:

```go
_, err := r.db.Exec(
    `UPDATE users SET is_active = false WHERE user_id = ANY($1)`,
    pq.Array(userIDs),
)
```

### 2. Переназначение ревьювера

**Проблема**: При переназначении нужно было учесть, что новый ревьювер не должен быть уже назначен на этот PR.

**Решение**: Добавил фильтрацию уже назначенных ревьюверов из списка кандидатов перед выбором нового:

```go
filtered := make([]*models.User, 0)
for _, candidate := range candidates {
    isAlreadyAssigned := false
    for _, assignedReviewerID := range pr.AssignedReviewers {
        if candidate.UserID == assignedReviewerID && candidate.UserID != oldUserID {
            isAlreadyAssigned = true
            break
        }
    }
    if !isAlreadyAssigned {
        filtered = append(filtered, candidate)
    }
}
```

### 3. Идемпотентность merge

**Проблема**: Нужно было обеспечить, чтобы при повторном merge не перезаписывалось время первого merge.

**Решение**: Использовал `COALESCE(merged_at, NOW())` в SQL, чтобы устанавливать `merged_at` только если оно еще не установлено.

### 4. Docker Compose команды

**Проблема**: Разные версии Docker используют `docker compose` (новый) или `docker-compose` (старый).

**Решение**: Добавил автоматическое определение в Makefile:

```makefile
DOCKER_COMPOSE_CMD := $(shell which docker-compose 2>/dev/null)
ifeq ($(DOCKER_COMPOSE_CMD),)
    DOCKER_COMPOSE_CMD := docker compose
endif
```

## Допущения

1. При создании команды проверяется, что команда с таким именем еще не существует.

2. При создании команды пользователи создаются или обновляются (ON CONFLICT DO UPDATE). Это означает, что пользователь может быть перемещен в другую команду.

3. Операции создания команды и PR выполняются в транзакциях.
