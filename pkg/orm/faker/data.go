package faker

var firstNames = []string{
	"Alex", "Aria", "Brian", "Chloe", "David", "Elena", "Felix", "Grace",
	"Hadi", "Indra", "Julia", "Kevin", "Luna", "Maya", "Nathan", "Olivia",
	"Putri", "Rian", "Sarah", "Tariq", "Uma", "Victor", "Wulan", "Xavier",
	"Yusuf", "Zoe", "Aditya", "Budi", "Citra", "Dian", "Eka", "Fajar",
	"Gita", "Hendra", "Intan", "Joko", "Kartika", "Lestari", "Mega", "Nugroho",
	"Pratama", "Rahmat", "Sari", "Tri", "Utami", "Wahyu", "Yanto", "Zainal",
	"Liam", "Noah", "Oliver", "James", "Elijah", "William", "Henry", "Lucas",
	"Benjamin", "Theodore", "Emma", "Charlotte", "Amelia", "Sophia", "Isabella", "Ava",
	"Mia", "Evelyn", "Harper", "Camila",
}

var lastNames = []string{
	"Anderson", "Bennett", "Chen", "Davis", "Evans", "Fischer", "Garcia", "Harris",
	"Iskandar", "Johnson", "Kusuma", "Lee", "Miller", "Nasution", "O'Connor", "Patel",
	"Qureshi", "Ramirez", "Santoso", "Tan", "Utomo", "Vargas", "Wijaya", "Xu",
	"Yamamoto", "Zimmerman", "Saputra", "Wibowo", "Hidayat", "Setiawan", "Prasetyo", "Siregar",
	"Simanjuntak", "Kurniawan", "Suryono", "Handoko", "Pratama", "Permana", "Gunawan", "Pangestu",
	"Smith", "Brown", "Taylor", "Wilson", "Clark", "Hall", "Young", "Allen",
	"Wright", "King", "Scott", "Green", "Baker", "Adams", "Nelson", "Carter",
}

var domains = []string{
	"example.com", "krewire.com", "cloud.io", "techcorp.net", "devstudio.org",
	"nexusapp.io", "alphaflow.dev", "synthetix.co", "byteforge.io", "datadock.com",
}

var freeEmailDomains = []string{
	"gmail.com", "yahoo.com", "hotmail.com", "outlook.com", "proton.me", "icloud.com",
}

var tlds = []string{"com", "org", "net", "io", "dev", "co", "id", "ai", "app"}

var streetNames = []string{
	"Sudirman", "Thamrin", "Gatot Subroto", "Rasuna Said", "Diponegoro", "Asia Afrika",
	"Main Street", "Market Street", "Broad Street", "Maple Avenue", "Oak Street", "Pine Street",
	"Elm Street", "Washington Boulevard", "Lincoln Avenue", "Sunset Boulevard", "Park Avenue",
	"Fifth Avenue", "Mission Street", "King Street", "Queen Street", "Victoria Road",
}

var cities = []string{
	"Jakarta", "Bandung", "Surabaya", "Yogyakarta", "Semarang", "Medan", "Denpasar", "Makassar",
	"Singapore", "Tokyo", "Seoul", "London", "Berlin", "San Francisco", "New York", "Seattle",
	"Toronto", "Sydney", "Amsterdam", "Stockholm", "Austin", "Boston", "Chicago", "Denver",
}

var states = []string{
	"DKI Jakarta", "Jawa Barat", "Jawa Timur", "DI Yogyakarta", "Jawa Tengah", "Bali", "Sumatera Utara",
	"California", "New York", "Washington", "Texas", "Illinois", "Massachusetts", "Colorado", "Ontario",
}

var countries = []string{
	"Indonesia", "Singapore", "Japan", "South Korea", "United States", "United Kingdom",
	"Germany", "Netherlands", "Australia", "Canada", "Sweden", "Switzerland", "France",
}

var companyPrefixes = []string{
	"Apex", "Acro", "Blue", "Byte", "Cloud", "Cyber", "Data", "Echo",
	"Flux", "Hyper", "Infra", "Kinetik", "Meta", "Nexus", "Omni", "Pulse",
	"Quantum", "Rapid", "Synapse", "Terra", "Ultra", "Vertex", "Wave", "Zenith",
}

var companySuffixes = []string{
	"Labs", "Tech", "Technologies", "Systems", "Networks", "Software", "Solutions",
	"Digital", "Dynamics", "Ventures", "Holdings", "Group", "Global", "Studios", "Media",
}

var jobTitles = []string{
	"Software Engineer", "Senior Backend Developer", "Fullstack Engineer", "DevOps Specialist",
	"Cloud Architect", "Site Reliability Engineer", "Product Manager", "Engineering Manager",
	"Security Analyst", "Data Engineer", "UI/UX Designer", "Technical Lead", "QA Automation Engineer",
	"Systems Architect", "Machine Learning Engineer", "Solutions Consultant",
}

var departments = []string{
	"Engineering", "Infrastructure", "Product", "Design", "Quality Assurance",
	"Security & Compliance", "Operations", "Finance", "Customer Success", "Marketing",
}

var productAdjectives = []string{
	"Ergonomic", "Intelligent", "Autonomous", "High-Performance", "Ultra-Light", "Durable",
	"Seamless", "Scalable", "Robust", "Precision", "Modular", "Adaptive", "Next-Gen",
}

var productMaterials = []string{
	"Aluminum", "Titanium", "Carbon Fiber", "Ceramic", "Polymer", "Steel", "Glass", "Bamboo",
}

var productTypes = []string{
	"Keyboard", "Mouse", "Monitor", "Headset", "Docking Station", "Router", "Microphone",
	"Webcam", "Desk Pad", "Laptop Stand", "Power Hub", "Storage Array", "Server Node",
}

var loremWords = []string{
	"alias", "consequatur", "aut", "perferendis", "sit", "voluptatem", "accusantium", "doloremque",
	"laudantium", "totam", "rem", "aperiam", "eaque", "ipsa", "quae", "ab",
	"illo", "inventore", "veritatis", "et", "quasi", "architecto", "beatae", "vitae",
	"dicta", "sunt", "explicabo", "nemo", "enim", "ipsam", "voluptatem", "quia",
	"voluptas", "aspernatur", "aut", "odit", "fugit", "sed", "consequuntur", "magni",
	"dolores", "eos", "qui", "ratione", "voluptatem", "sequi", "nesciunt", "neque",
	"porro", "quisquam", "est", "dolorem", "ipsum", "dolor", "amet", "consectetur",
	"adipisci", "velit", "eius", "tempora", "incidunt", "labore", "dolore", "magnam",
	"aliquam", "quaerat", "voluptatem", "ad", "minima", "veniam", "nostrum", "exercitationem",
	"ullam", "corporis", "suscipit", "laboriosam", "nisi", "aliquid", "ex", "commodi",
}

var userAgents = []string{
	"Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
	"Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/128.0.0.0 Safari/537.36",
	"Mozilla/5.0 (X11; Ubuntu; Linux x86_64; rv:130.0) Gecko/20100101 Firefox/130.0",
	"Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:130.0) Gecko/20100101 Firefox/130.0",
}
