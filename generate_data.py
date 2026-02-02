import random
import json

names = [
    "Adam", "Bela", "Cecil", "David", "Elemer", "Ferenc", "Gabor", "Hugo", "Istvan", "Janos",
    "Kata", "Luca", "Mari", "Nora", "Orsolya", "Panna", "Reka", "Sara", "Timea", "Zsuzsanna",
    "Peter", "Laszlo", "Zoltan", "Attila", "Tamas", "Andras", "Sandor", "Jozsef", "Gyorgy"
]
surnames = [
    "Kovacs", "Nagy", "Szabo", "Toth", "Varga", "Kiss", "Molnar", "Nemeth", "Farkas", "Balog",
    "Papp", "Takacs", "Juhasz", "Lakatos", "Meszaros", "Simon", "Kelemen", "Hegedus"
]
departments = ["ENG", "MGT", "DSG", "HR", "OPS", "FIN", "SALES"]
roles = {
    "ENG": ["Developer", "Senior Developer", "Architect", "QA Engineer"],
    "MGT": ["Project Manager", "Product Owner", "Team Lead"],
    "DSG": ["UI Designer", "UX Researcher", "Graphic Designer"],
    "HR": ["Recruiter", "HR Specialist"],
    "OPS": ["Sysadmin", "DevOps Engineer"],
    "FIN": ["Accountant", "Financial Analyst"],
    "SALES": ["Sales Rep", "Account Manager"]
}
tags = ["go", "java", "python", "js", "sql", "aws", "docker", "kubernetes", "react", "vue"]
certs = ["PMP", "AWS", "CKA", "CISSP", "SCRUM"]

print("INSERT INTO personnel (name, email, department, salary, is_valid, is_active, data) VALUES")

entries = []
for i in range(100):
    first = random.choice(names)
    last = random.choice(surnames)
    full_name = f"{last} {first}"
    email = f"{first.lower()}.{last.lower()}@example.com"
    dept = random.choice(departments)
    salary = random.randint(45000, 150000)
    is_valid = random.choice([True] * 9 + [False]) # 90% valid
    is_active = random.choice([True] * 9 + [False])
    
    role = random.choice(roles[dept])
    data = {
        "role": role,
        "experience": random.randint(1, 20)
    }
    
    if dept == "ENG":
        data["tags"] = random.sample(tags, k=random.randint(1, 4))
    if dept == "MGT" and random.random() > 0.5:
        data["certifications"] = random.sample(certs, k=random.randint(1, 2))
    
    entries.append(f"('{full_name}', '{email}', '{dept}', {salary}, {str(is_valid).lower()}, {str(is_active).lower()}, '{json.dumps(data)}')")

print(",\n".join(entries))
print(" ON CONFLICT (id) DO NOTHING;")
