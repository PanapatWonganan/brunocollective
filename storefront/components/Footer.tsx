import Link from "next/link";
import styles from "./Footer.module.css";

export default function Footer() {
  return (
    <footer id="contact" className={styles.footer}>
      <div className="wrap">
        <div className={styles.grid}>
          <div className={styles.brandBlock}>
            <div className={styles.mark}>BC</div>
            <div className={styles.name}>Bruno Collective</div>
            <p className={styles.pitch}>
              A small Thai house, quietly making considered clothing from a love
              of fine cloth and understated luxury. Designed &amp; made in Thailand.
            </p>
          </div>

          <div>
            <h5 className={styles.h5}>The House</h5>
            <ul className={styles.ul}>
              <li><Link href="/story">Our Story</Link></li>
              <li><Link href="/story#atelier">The Atelier</Link></li>
              <li><Link href="/lookbook">Lookbook</Link></li>
              <li><Link href="/journal">Journal</Link></li>
              <li><Link href="/member">Membership</Link></li>
            </ul>
          </div>

          <div>
            <h5 className={styles.h5}>Service</h5>
            <ul className={styles.ul}>
              <li><Link href="/shop">The Collection</Link></li>
              <li><Link href="/service#appointment">Private Appointment</Link></li>
              <li><Link href="/service#care">Care &amp; Repair</Link></li>
              <li><Link href="/service#shipping">Shipping &amp; Exchange</Link></li>
              <li><Link href="/service#contact">Contact</Link></li>
            </ul>
          </div>

          <div className={styles.boutiques} id="boutiques">
            <h5 className={styles.h5}>Atelier</h5>
            <ul className={styles.ul}>
              <li><span className={styles.city}>The Studio</span><span className={styles.addr}>Thailand · By appointment · นัดหมายล่วงหน้า</span></li>
              <li><span className={styles.city}>Online</span><span className={styles.addr}>Shipping across Thailand · จัดส่งทั่วไทย</span></li>
            </ul>
          </div>

          <div>
            <h5 className={styles.h5}>Correspondence</h5>
            <ul className={styles.ul}>
              <li><a href="mailto:hello@brunocollective.co">hello@brunocollective.co</a></li>
              <li><Link href="/service#appointment">Private Appointment · นัดหมาย</Link></li>
              <li style={{ marginTop: 18 }}><a href="mailto:hello@brunocollective.co?subject=Press%20Enquiry">Press Enquiries</a></li>
              <li><a href="mailto:hello@brunocollective.co?subject=Wholesale">Wholesale</a></li>
            </ul>
          </div>
        </div>

        <div className={styles.bottom}>
          <div>© 2026 Bruno Collective — All rights reserved.</div>
          <div className={styles.social}>
            <a href="https://www.instagram.com/bruno.collective/" target="_blank" rel="noreferrer">Instagram</a>
            <Link href="/journal">Journal</Link>
            <a href="https://lin.ee/tT3JcJX" target="_blank" rel="noreferrer">LINE</a>
          </div>
          <div>Designed &amp; made in Thailand</div>
        </div>
      </div>
    </footer>
  );
}
