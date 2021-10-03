#pragma once


namespace person {

struct Person {
  int age;
  float health;
};

inline Person NewPerson(int age, float health) {
  return Person { age, health };
}

inline int Age(Person p) {
  return p.age;
}

inline float Health(Person p) {
  return p.health;
}

inline void Grow(Person *p) {
  p->age++;
}

}