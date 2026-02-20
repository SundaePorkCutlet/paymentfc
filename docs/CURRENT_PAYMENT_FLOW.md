# PAYMENTFC — 현재(원래) 결제 플로우 (수정 전 기준)

강의에서 배치 방식으로 수정하기 **전** 상태를 기록해 둔 문서.  
README에 "원래는 ~였고, 이렇게 수정했다" 적을 때 참고용.

---

## 현재 방식 (Real-time invoice creation)

1. **유저 체크아웃** → ORDERFC가 `order.created` 이벤트 발행 (Kafka)
2. **PAYMENTFC**가 `order.created`를 **바로 컨슈밍**
3. **수신 즉시** `xenditUsecase.CreateInvoice(event)` 호출 → Xendit API로 인보이스 생성
4. 유저는 **바로 결제창/결제 링크**를 받을 수 있음

### 관련 코드 위치

- **main.go**  
  - `kafka.StartOrderConsumer(..., func(event) { xenditUsecase.CreateInvoice(...) })`  
  - 이벤트 수신 시 곧바로 인보이스 생성
- **kafka/order_consumer.go**  
  - `order.created` 구독, 메시지 수신 시 위 핸들러 호출
- **스케줄러 (scheduler_service.go)**  
  - **인보이스 생성**이 아니라 **이미 만든 인보이스의 결제 완료 여부**를 10분마다 Xendit에 물어보고, PAID면 `ProcessPaymentSuccess` 호출 (웹훅 미도착 대비)

### DB

- **payments**: 인보이스 생성 시/웹훅·스케줄러로 결제 완료 시 저장
- **payment_requests**: 테이블만 있음 (배치 플로우에서 사용 예정)

---

## 수정 후 (강의에서 적용할 New condition) — 요약만

- `order.created` 수신 시 → **인보이스 생성 X**, `payment_requests`에 **저장만**
- **별도 배치**가 `payment_requests`를 읽어서 인보이스 생성 (Xendit 호출)
- Pros: Xendit 점검/장애 시 실행 보류 가능, 이벤트 유실 없이 보관  
- Cons: 실시간 아님, 유저는 배치가 돌 때까지 대기

---

수정 반영 후 README에 "원래 방식은 위와 같았고, 배치 방식으로 이렇게 수정했다" 식으로 추가하면 됨.
