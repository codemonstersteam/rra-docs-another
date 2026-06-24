# intent — assess

Оценить качество документации **произвольного** git-репозитория для четырёх
JTBD-потребителей: выдать четыре независимых score + конкретные пробелы с
`file:line`, не предполагая рациональную дисциплину.

## Контекст

- Контракт: [`api-specification/cli.md`](../../../api-specification/cli.md) +
  [`report.schema.json`](../../../api-specification/report.schema.json).
- Исполняемая спецификация: `component-tests/features/*.feature` (гейт E1 в main).
- Концепция слоёв L1–L6 и JTBD: [`CONCEPT.md`](../../../CONCEPT.md).
- Архитектура: [`c4.md`](c4.md) (C2/C3 + системные use case по Коберну),
  [`infrastructure.md`](infrastructure.md), сквозной выход — [`egress.md`](egress.md).
- Один внешний вход = одна подкоманда = один slice. Семь слайсов (S1–S7) +
  поздний S8 (`drift --semantic`, тир L6c).

## Границы

- НЕ проверяет соответствие дисциплине (это гейт `rra-docs`).
- НЕ усредняет JTBD: четыре score остаются раздельными.
- ИИ только на L5 (`fitness`) и опциональном L6c (`drift --semantic`).
- Дешёвое-первым: L1/L3/L4/L6a (ноль интеграций) → L2 (subprocess) → L5/L6c (LLM).
