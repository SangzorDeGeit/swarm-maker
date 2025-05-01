package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
)

type Monster struct {
	Name                   string
	Armor_class            []ArmorClass
	Hit_points             int
	Strength               int
	Dexterity              int
	Constitution           int
	Intelligence           int
	Wisdom                 int
	Charisma               int
	Proficiencies          []Proficiencies
	Damage_vulnerabilities []string
	Damage_resistances     []string
	Damage_immunities      []string
	Condition_immunities   []Condition
	Proficiency_bonus      int
	Actions                []Action
}

type ArmorClass struct {
	Value int
}

type Proficiencies struct {
	Value       int
	Proficiency Proficiency
}

type Proficiency struct {
	Index string
}

type Condition struct {
	Name string
}

type Action struct {
	Attack_bonus int
	Damage       []Damage
}

type Damage struct {
	Damage_type Damage_type
	Damage_dice string
}

type Damage_type struct {
	Name string
}

func get_mod(score int) int {
	raw_value := score - 10
	if raw_value < 0 {
		absolute := math.Abs(float64(raw_value))
		mod := math.Ceil(absolute / 2)
		return int(-1 * mod)
	} else {
		mod := math.Floor(float64(raw_value) / 2)
		return int(mod)
	}
}

func parse_damage(damage_dice string) (int, string, int) {
	binding := strings.Split(damage_dice, "d")
	amount, err := strconv.Atoi(binding[0])
	if err != nil {
		log.Fatal(err)
	}
	if len(binding) == 1 {
		return 0, "", amount
	}
	binding = strings.Split(binding[1], "+")
	die_count := binding[0]
	var bonus int
	if len(binding) > 1 {
		bonus, err = strconv.Atoi(binding[1])
		if err != nil {
			log.Fatal(err)
		}
	}
	return amount, die_count, bonus

}

func print_save(mod_name string, modifier int) {
	if modifier > 0 {
		fmt.Printf("%s: +%d\n", mod_name, modifier)
	} else if modifier < 0 {
		fmt.Printf("%s: %d\n", mod_name, modifier)
	} else {
		fmt.Printf("%s:  %d\n", mod_name, modifier)
	}
}

func main() {
	fmt.Print("Please give the monster name: ")
	scanner := bufio.NewScanner(os.Stdin)
	scanner.Scan()
	err := scanner.Err()
	if err != nil {
		log.Fatal(err)
	}
	monster := scanner.Text()
	// format input for proper API input
	monster = strings.ToLower(monster)
	monster = strings.ReplaceAll(monster, " ", "-")

	fmt.Print("Please give amount of monsters in the swarm. The amount should be a multiple of- and minimum of 10: ")
	scanner.Scan()
	err = scanner.Err()
	if err != nil {
		log.Fatal(err)
	}
	n := scanner.Text()
	amount, err := strconv.Atoi(n)
	if err != nil {
		fmt.Println("Please input a whole number")
		log.Fatal(err)
	}
	if amount < 10 || amount%10 != 0 {
		fmt.Println("The amount in the swarm should be a multiple of- and minimally 10")
	}
	segments := amount / 10

	// Get the data for the monster
	request_url := fmt.Sprintf("https://www.dnd5eapi.co/api/2014/monsters/%s", monster)
	response, err := http.Get(request_url)
	if err != nil {
		log.Fatal(err)
	}
	response_data, err := io.ReadAll(response.Body)
	if err != nil {
		log.Fatal(err)
	}
	if string(response_data) == "Not Found" {
		fmt.Println("Monster does not exist")
		os.Exit(0)
	}
	var monster_stats Monster
	err = json.Unmarshal(response_data, &monster_stats)
	if err != nil {
		log.Fatal(err)
	}

	// Extract variables from response
	ac := monster_stats.Armor_class[0].Value

	// calculate saving throw bonusses
	var str_save int
	var dex_save int
	var con_save int
	var int_save int
	var wis_save int
	var cha_save int
	for _, proficiencies := range monster_stats.Proficiencies {
		if proficiencies.Proficiency.Index == "saving-throw-str" {
			str_save = proficiencies.Value
		}
		if proficiencies.Proficiency.Index == "saving-throw-dex" {
			dex_save = proficiencies.Value
		}
		if proficiencies.Proficiency.Index == "saving-throw-con" {
			con_save = proficiencies.Value
		}
		if proficiencies.Proficiency.Index == "saving-throw-int" {
			int_save = proficiencies.Value
		}
		if proficiencies.Proficiency.Index == "saving-throw-wis" {
			wis_save = proficiencies.Value
		}
		if proficiencies.Proficiency.Index == "saving-throw-cha" {
			cha_save = proficiencies.Value
		}
	}

	if str_save == 0 {
		str_save = get_mod(monster_stats.Strength)
	}
	if dex_save == 0 {
		dex_save = get_mod(monster_stats.Dexterity)
	}
	if con_save == 0 {
		con_save = get_mod(monster_stats.Constitution)
	}
	if int_save == 0 {
		int_save = get_mod(monster_stats.Intelligence)
	}
	if wis_save == 0 {
		wis_save = get_mod(monster_stats.Wisdom)
	}
	if cha_save == 0 {
		cha_save = get_mod(monster_stats.Charisma)
	}

	// calculate hp related
	hp_per_unit := math.Round(float64(monster_stats.Hit_points) / 5)
	if hp_per_unit == 0 {
		hp_per_unit = 1
	}
	total_hp := amount * int(hp_per_unit)
	hp_per_segment := int(hp_per_unit * 10)
	// calculate attacks related
	var to_hit int
	var dice_count int
	var dice string
	var damage_mod int
	var damage_type string
	for _, attack := range monster_stats.Actions {
		if attack.Attack_bonus != 0 {
			to_hit = attack.Attack_bonus
		}
		if len(attack.Damage) != 0 {
			dice_count, dice, damage_mod = parse_damage(attack.Damage[0].Damage_dice)
			damage_type = attack.Damage[0].Damage_type.Name
			break
		}
	}
	to_hit = to_hit + (2 * (segments - 1))
	total_dice_count := segments * dice_count
	total_damage_mod := segments * damage_mod

	// Print everything
	fmt.Printf("\n==========================================\n")
	fmt.Printf("Ac: %d\n", ac)
	fmt.Printf("Total hp: %d / %d\n", total_hp, hp_per_segment)
	fmt.Printf("to hit: %d / 2\n", to_hit)
	if total_dice_count == 0 {
		fmt.Printf("Damage: %d / %d %s\n", total_damage_mod, damage_mod, damage_type)
	} else if total_damage_mod == 0 {
		fmt.Printf("Damage: %dd%s / %dd%s %s\n", total_dice_count, dice, dice_count, dice, damage_type)
	} else {
		fmt.Printf("Damage: %dd%s+%d / %dd%s+%d %s\n", total_dice_count, dice, total_damage_mod, dice_count, dice, damage_mod, damage_type)
	}
	fmt.Printf("\nSaving throw bonusses:\n")
	print_save("Str", str_save)
	print_save("Dex", dex_save)
	print_save("Con", con_save)
	print_save("Int", int_save)
	print_save("Wis", wis_save)
	print_save("Cha", cha_save)

	for i, c := range monster_stats.Condition_immunities {
		if i == 0 {
			fmt.Printf("\nCondition immunities: %s", c.Name)
		} else {
			fmt.Printf(", %s", c.Name)
		}
	}
	for i, c := range monster_stats.Damage_vulnerabilities {
		if i == 0 {
			fmt.Printf("\nDamage vulnerabilities: %s", c)
		} else {
			fmt.Printf(", %s", c)
		}
	}
	for i, c := range monster_stats.Damage_resistances {
		if i == 0 {
			fmt.Printf("\nDamage resistances: %s", c)
		} else {
			fmt.Printf(", %s", c)
		}
	}
	for i, c := range monster_stats.Damage_immunities {
		if i == 0 {
			fmt.Printf("\nDamage immunities: %s", c)
		} else {
			fmt.Printf(", %s", c)
		}
	}
	fmt.Printf("\n==========================================\n")

}
