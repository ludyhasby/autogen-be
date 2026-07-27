import streamlit as st 
import numpy as np
import pandas as pd 
import datetime
from time import sleep
import altair as alt
from helpers.convert import prev_three_four, decode_month, fix_blth_1, countPeriod, WBPOrLWBP
from helpers.reader import read_excel, read_csv_try, read_csv
from helpers.errorHandler import errorHandling
from helpers.processing import safe_mean, fmt_num, v_drop_lp, i_loss_lp, freeze_lp
from helpers.style import highlighter
import time
import io
from dotenv import load_dotenv
import os 

load_dotenv()
user_pass = os.getenv("APP_PASS")
user_pass2 = os.getenv("APP_PASS2")

st.set_page_config(page_title="Auto Dashboarding V2", 
                   page_icon = "https://github.com/user-attachments/assets/15349020-096c-4c47-8022-749a6b0593c1", 
                   layout = "wide", 
                   initial_sidebar_state = "auto")

st.markdown("""
    <style>
    /* Force light mode */
    body, html, .stApp {
        background-color: white !important;
        color: black !important;
    }

    html, body, [class^="css"] {
        color: black !important;
        background-color: white !important;
    }

    /* Hide Streamlit branding */
    header {visibility: hidden;}
    [data-testid="stSidebar"] {
        display: none;
    }
    /* Custom Navbar */
    .navbar {
        background-color: #2596be;
        padding: 20px 30px;
        display: flex;
        justify-content: space-between;
        align-items: center;
        border-bottom: 1px solid #ccc;
    }
    .navbar-logo {
        display: flex;
        align-items: center;
        font-weight: bold;
        color: #4B2E1E;
        font-size: 22px;
    }
    .navbar-logo img {
        height: 35px;
        margin-right: 10px;
    }
    .navbar-menu a {
        margin: 0 15px;
        text-decoration: none;
        color: #111;
        font-weight: 600;
    }
    .navbar-menu a:hover {
        color: #a16736;
    }
    .navbar-menu a.active {
        font-size: 20px;
        font-weight: bold;
    }
    .mirror-link {
    color: red;
    }
    /* Footer Styling */
    .footer {
        background-color: #2596be;
        color: #4B2E1E;
        font-family: sans-serif;
        padding: 40px 20px;
        position: relative;
    }
    .footer-container {
        display: flex;
        justify-content: space-between;
        flex-wrap: wrap;
    }
    .footer-column {
        flex: 1;
        min-width: 200px;
        margin: 10px;
    }
    .footer-column h3 {
        color: black;
        margin-bottom: 10px;
    }
    .footer-column ul {
        list-style: none;
        padding: 0;
    }
    .footer-column ul li {
        margin-bottom: 8px;
    }
    .footer-bottom {
        margin-top: 10px;
        text-align: left;
        font-size: 14px;
        color: #ccc;
    }
    </style>
""", unsafe_allow_html=True)
# AMR Worker
def dashboarding_amr(df_amr):
    # "POWER_FACTOR_L1","POWER_FACTOR_L2","POWER_FACTOR_L3"
    df_amr.rename({'LOCATION_CODE': 'IDPEL'}, axis=1, inplace=True)
    df_amr['IDPEL'] = df_amr['IDPEL'].astype(str)
    period_data_y = str(df_amr.loc[0, "PERIODE"])
    period_data = datetime.datetime(int(period_data_y[:4]), int(period_data_y[4:]), 1)
    datetime_data_month = period_data.strftime("%B")
    datetime_data_year = period_data.strftime("%Y")
    
    st.markdown(f"""<h2>Dashboard Target Operasi P2TL AMR Periode {datetime_data_month} {datetime_data_year}</h2>""", unsafe_allow_html=True)
    df_amr['JAM'] = df_amr['READ_DATE'].dt.hour
    df_amr['WAKTU'] = np.where((df_amr['JAM'] < 6) | (df_amr['JAM'] > 18), "MALAM", "SIANG")
    df_amr = df_amr[df_amr['LOCATION_TYPE']=="CUSTOMER"] # Filter CUSTOMER
    # Handle duplicate
    df_amr = df_amr.sort_values(by='READ_DATE', ascending=False).drop_duplicates(subset=['IDPEL', 'WAKTU']).reset_index(drop=True)
    n_data = len(df_amr)
    n_idpel = len(df_amr['IDPEL'].unique())

    tegangan_maks = df_amr[['VOLTAGE_L1', 'VOLTAGE_L2', 'VOLTAGE_L3']].max(axis=1).astype(float)
    df_amr['PENG'] = np.where(tegangan_maks> 70, "TR", "TM")

    try: 
        df_amr['POWER'] = df_amr['POWER'].astype(int)
    except Exception as e:
        df_amr['POWER'] = df_amr['POWER'].str.replace(',', '').astype(int)
    df_amr.rename({'NAMA_PELANGGAN':'NAMA', 'TARIFF':'TARIF', 'POWER':'DAYA (VA)'}, axis=1, inplace=True)

    # Jenis Pengukuran
    df_amr['Jenis Pengukuran'] = np.where(df_amr['DAYA (VA)'] < 53000, "Langsung", "Tak Langsung")
    df_amr['PHASE'] = np.where(
        (df_amr['DAYA (VA)'] > 11000) | (df_amr['DAYA (VA)'] == 10600) | (df_amr['DAYA (VA)'] == 6600),
        3,1)
    df_amr['BILL_REFF_KWH'] = df_amr['BILL_REFF_KWH'].astype(int)

    # benerin to float 
    float_column = [
        'VOLTAGE_L1', 'VOLTAGE_L2', 'VOLTAGE_L3',
        'VOLTAGE_ANGLE_L1', 'VOLTAGE_ANGLE_L2', 'VOLTAGE_ANGLE_L3', 
        'CURRENT_L1', 'CURRENT_L2', 'CURRENT_L3',
        'CURRENT_N', 'ACTIVE_POWER_L1', 'ACTIVE_POWER_L2',
        'ACTIVE_POWER_L3', 'ACTIVE_POWER_TOTAL', 'KWH_ABS_TOTAL',
        'CURRENT_ANGLE_L1', 'CURRENT_ANGLE_L2', 'CURRENT_ANGLE_L3', 
        'APPARENT_POWER_L1', 'APPARENT_POWER_L2', 'APPARENT_POWER_L3'
    ]
    for col in float_column:
        df_amr[col] = pd.to_numeric(
            df_amr[col].astype(str).str.replace(',', '.', regex=False),
            errors='coerce'
        )

    # COS PHI MEASUREMENT
    df_amr["POWER_FACTOR_L1"] = np.where(
        df_amr["APPARENT_POWER_L1"] != 0,
        df_amr["ACTIVE_POWER_L1"] / df_amr["APPARENT_POWER_L1"],
        np.cos(np.radians(df_amr["CURRENT_ANGLE_L1"] - df_amr["VOLTAGE_ANGLE_L1"]))
    )
    df_amr["POWER_FACTOR_L2"] = np.where(
        df_amr["APPARENT_POWER_L2"] != 0,
        df_amr["ACTIVE_POWER_L2"] / df_amr["APPARENT_POWER_L2"],
        np.cos(np.radians(df_amr["CURRENT_ANGLE_L2"] - df_amr["VOLTAGE_ANGLE_L2"]))
    )
    df_amr["POWER_FACTOR_L3"] = np.where(
        df_amr["APPARENT_POWER_L3"] != 0,
        df_amr["ACTIVE_POWER_L3"] / df_amr["APPARENT_POWER_L3"],
        np.cos(np.radians(df_amr["CURRENT_ANGLE_L3"] - df_amr["VOLTAGE_ANGLE_L3"]))
    )

    with st.expander("Setting Parameter 📐"):
        st.write("Operasi Logika yang digunakan disini adalah OR. Dengan demikian, indikator yang sesuai dengan salah satu spesifikasi aturan tersebut akan di highlight berwarna hijau cerah dan berkontribusi pada perhitungan potensi TO.")
        # FILTER
        co1, co2, co3, co4, co5, co6, co7 = st.columns(7)
        with co1: 
            st.markdown(
                "<h5>Tegangan Drop</h5>"
                "<p>Tegangan drop secara sederhana dapat terjadi ketika <b>salah satu</b> L1, L2, L3 ber tegangan kecil <b>dan</b> bernilai positif <b>dan</b> ber arus besar.</p>",
                unsafe_allow_html=True
            )
            st.markdown("------")
            batas_atas_v_tm = st.number_input(label="Set Batas Atas Tegangan Menengah (tm) | tm < ", value=56.0)        
            batas_atas_v_tr = st.number_input(label="Set Batas Atas Tegangan Rendah (tr) | tr < ", value=180.0)                
            batas_bawah_arus_tm = st.number_input(label="Set Batas Bawah Arus Besar tm | I(tm) > ", value=0.5)   
            batas_bawah_arus_tr = st.number_input(label="Set Batas Bawah Arus Besar tr | I(tr) > ", value=0.5)   
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            # Kondisi untuk TM
            condition_tm = (
                ((df_amr['VOLTAGE_L1'] < batas_atas_v_tm) & (df_amr['VOLTAGE_L1'] >= 0) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tm)) |
                ((df_amr['VOLTAGE_L2'] < batas_atas_v_tm) & (df_amr['VOLTAGE_L2'] >= 0) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tm)) |
                ((df_amr['VOLTAGE_L3'] < batas_atas_v_tm) & (df_amr['VOLTAGE_L3'] >= 0) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tm))
            )
            # Kondisi untuk TR
            condition_tr = (
                ((df_amr['VOLTAGE_L1'] < batas_atas_v_tr) & (df_amr['VOLTAGE_L1'] >= 0) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tr)) |
                ((df_amr['VOLTAGE_L2'] < batas_atas_v_tr) & (df_amr['VOLTAGE_L2'] >= 0) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tr)) |
                ((df_amr['VOLTAGE_L3'] < batas_atas_v_tr) & (df_amr['VOLTAGE_L3'] >= 0) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tr))
            )

            # Gabungkan kondisi berdasarkan tipe PENG
            condition_v_drop = (mask_tm & condition_tm) | (mask_tr & condition_tr)
            df_amr["v_drop"] = condition_v_drop
            condition_v_drop_tak_langsung = condition_v_drop & (df_amr['DAYA (VA)'] >= 53000)
            condition_v_drop_langsung = condition_v_drop & (df_amr['DAYA (VA)'] < 53000)
        
        with co2:
            st.markdown(
                "<h5>Tegangan Hilang</h5>"
                "<p>Tegangan hilang adalah data dengan PHASE 3 yang tegangannya hilang <b>dan</b> arus (I) lebih dari ambang <b>pada salah satu</b> L1, L2, atau L3. </p>",
                unsafe_allow_html=True
            )
            st.markdown("------")
            tegangan_hilang_tm = st.number_input(label="Nilai Tegangan Menengah Hilang (tm) | tm = ", value=0.0)
            tegangan_hilang_tr = st.number_input(label="Nilai Tegangan Rendah Hilang (tr) | tr = ", value=0.0)
            batas_bawah_arus_tm_2 = st.number_input(label="Set Batas Bawah Arus Besar tm | I(tm) > ", value=-1.0)   
            batas_bawah_arus_tr_2 = st.number_input(label="Set Batas Bawah Arus Besar tr | I(tr) > ", value=-1.0)   
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            # Kondisi untuk TM
            condition_tm = (
                (df_amr['PHASE']==3) & (
                    ((df_amr['VOLTAGE_L1'] == tegangan_hilang_tm) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tm_2)) |
                    ((df_amr['VOLTAGE_L2'] == tegangan_hilang_tm) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tm_2)) |
                    ((df_amr['VOLTAGE_L3'] == tegangan_hilang_tm) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tm_2))
                )
            )
            # Kondisi untuk TR
            condition_tr = (
                (df_amr['PHASE']==3) & (
                    ((df_amr['VOLTAGE_L1'] == tegangan_hilang_tr) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tr_2)) |
                    ((df_amr['VOLTAGE_L2'] == tegangan_hilang_tr) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tr_2)) |
                    ((df_amr['VOLTAGE_L3'] == tegangan_hilang_tr) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tr_2))
                )
            )
            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["v_lost"] = ((mask_tm & condition_tm) | (mask_tr & condition_tr)).fillna(False)

        with co3: 
            st.markdown(
                "<h5>Cos Phi Kecil</h5>"
                "<p>Cos phi kecil atau besaran daya yang dipakai untuk kerja kecil <b>dan</b> pada salah satu L1, L2, atau L3 <b>dan</b> khusus tak langsung.</p>",
                unsafe_allow_html=True
            )
            st.markdown("------")
            batas_atas_cos_phi_tm = st.number_input(label="Set Batas Atas Cos Phi Kecil pada Tegangan Menengah (tm) | cos phi <= ", value=0.4)
            batas_atas_cos_phi_tr = st.number_input(label="Set Batas Atas Cos Phi Kecil pada Tegangan Rendah (tr) | cos phi <= ", value=0.4)
            batas_bawah_arus_tm_3 = st.number_input(label="Set Batas Bawah Arus Besar tm cos phi kecil | I(tm) > ", value=0.8)   
            batas_bawah_arus_tr_3 = st.number_input(label="Set Batas Bawah Arus Besar tr cos phi kecil | I(tr) > ", value=0.8)   
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            # Kondisi untuk TM
            condition_tm = (
                ((df_amr['POWER_FACTOR_L1'].abs() <= batas_atas_cos_phi_tm) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tm_3)) |
                ((df_amr['POWER_FACTOR_L2'].abs() <= batas_atas_cos_phi_tm) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tm_3)) |
                ((df_amr['POWER_FACTOR_L3'].abs() <= batas_atas_cos_phi_tm) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tm_3))
            )
            # Kondisi untuk TR
            condition_tr = (
                ((df_amr['POWER_FACTOR_L1'].abs() <= batas_atas_cos_phi_tr) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tr_3)) |
                ((df_amr['POWER_FACTOR_L2'].abs() <= batas_atas_cos_phi_tr) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tr_3)) |
                ((df_amr['POWER_FACTOR_L3'].abs() <= batas_atas_cos_phi_tr) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tr_3))
            )

            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["cos_phi_kecil"] = (((mask_tm & condition_tm) | (mask_tr & condition_tr)) & (df_amr['Jenis Pengukuran']=="Tak Langsung")).fillna(False)

        with co4: 
            st.markdown(
                "<h5>Arus Hilang</h5>"
                "<p>arus hilang terjadi jika arus nya kecil <b>dan</b> arusnya pernah besar <b>pada salah satu</b> L1, L2, atau L3. </p>",
                unsafe_allow_html=True
            )
            st.markdown("------")  
            batas_atas_arus_hilang_tm = st.number_input(label="Set Batas Atas arus hilang pada Tegangan Menengah (tm) | arus <= ", value=0.02)
            batas_atas_arus_hilang_tr = st.number_input(label="Set Batas Atas arus hilang pada Tegangan Rendah (tr) | arus <= ", value=0.02)
            batas_bawah_arus_maks_tm= st.number_input(label="Set Batas Bawah Arus Maksimum tm | I maks > ", value=1.0)   
            batas_bawah_arus_maks_tr = st.number_input(label="Set Batas Bawah Arus Maksimum tr | I maks >", value=1.0)   
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            arus_maks = df_amr[['CURRENT_L1', 'CURRENT_L2', 'CURRENT_L3']].max(axis=1)
            arus_mins = df_amr[['CURRENT_L1', 'CURRENT_L2', 'CURRENT_L3']].min(axis=1)
            # Kondisi untuk TM
            condition_tm = (
                ((df_amr['CURRENT_L1'] <= batas_atas_arus_hilang_tm) & (arus_maks > batas_bawah_arus_maks_tm)) |
                ((df_amr['CURRENT_L2'] <= batas_atas_arus_hilang_tm) & (arus_maks > batas_bawah_arus_maks_tm)) |
                ((df_amr['CURRENT_L3'] <= batas_atas_arus_hilang_tm) & (arus_maks > batas_bawah_arus_maks_tm))
            )
            # Kondisi untuk TR
            condition_tr = (
                ((df_amr['CURRENT_L1'] <= batas_atas_arus_hilang_tr) & (arus_maks > batas_bawah_arus_maks_tr)) |
                ((df_amr['CURRENT_L2'] <= batas_atas_arus_hilang_tr) & (arus_maks > batas_bawah_arus_maks_tr)) |
                ((df_amr['CURRENT_L3'] <= batas_atas_arus_hilang_tr) & (arus_maks > batas_bawah_arus_maks_tr))
            )

            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["arus_hilang"] = ((mask_tm & condition_tm) | (mask_tr & condition_tr)).fillna(False)

        with co5:
            st.markdown(
                "<h5>Arus Netral Lebih besar dari Arus Maksimum</h5>"
                "<p>kriterianya adalah arus netral lebih besar dari (arus maksimum - arus minimum + 5) <b>dan</b> lebih besar dari nilai arus netral yang ditentukan.</p>",
                unsafe_allow_html=True
            )
            st.warning("⚠️ Untuk Tipe Meter HXE310, AMETER300, MK10MI harap dilakukan evaluasi lebih lanjut")
            st.markdown("------")  
            batas_bawah_arus_netral_tm = st.number_input("Set Batas Bawah Arus Netral tm | I netral > ", value=1.0)
            batas_bawah_arus_netral_tr = st.number_input("Set Batas Bawah Arus Netral tr | I netral > ", value=10.0)
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            # Kondisi untuk TM
            condition_tm = (
                ((df_amr['CURRENT_N'] > (arus_maks-arus_mins+5)) & (df_amr['CURRENT_N'] > batas_bawah_arus_netral_tm))
            )
            # Kondisi untuk TR
            condition_tr = (
                ((df_amr['CURRENT_N'] > (arus_maks-arus_mins+5)) & (df_amr['CURRENT_N'] > batas_bawah_arus_netral_tr))
            )

            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["In_more_Imax"] = ((mask_tm & condition_tm) | (mask_tr & condition_tr)).fillna(False)

        with co6:
            # Logika di SHEET untuk kolom AH sepertinya masih kurang 
            st.markdown(
                "<h5>Over Current</h5>"
                "<p>Nilai arus Maks(L1, L2, L3) lebih besar dari nilai yang kami tentukan (untuk daya tertentu) + Nilai Arus Maks(L1, L2, L3) yang lebih besar dari nilai yang ditentukan user. </p>",
                unsafe_allow_html=True
            )
            st.markdown("------") 
            arus_maks_tm = st.number_input(label="Set Batas bawah Arus Maks (Pengukuran Tak Langsung) pada Tegangan Menengah (tm)", value=5.0)
            arus_maks_tr = st.number_input(label="Set Batas bawah Arus Maks (Pengukuran Tak Langsung) pada Tegangan Rendah (tr)", value=5.0)

            # pengukuran langsung
            daya_lansung = [450, 900, 1300, 2200, 3500, 4400, 5500, 6600, 7700, 10600, 11000, 13200, 16500, 23000, 33000, 41500]
            swither_over_current = {
                450: 2, 
                900: 4, 
                1300: 6, 
                2200: 10, 
                3500: 16, 
                4400: 20, 
                5500: 25, 
                6600: 10, 
                7700: 35, 
                10600: 16, 
                11000: 50, 
                13200: 20, 
                16500: 25, 
                23000: 35, 
                33000: 50, 
                41500: 63
            }
            swither_series = df_amr['DAYA (VA)'].map(swither_over_current)*1.4  # konversi daya → nilai arus ambang
            mask_nan_langsung = swither_series.isna() & (df_amr['DAYA (VA)'] < 53000)
            swither_series = swither_series.where(~mask_nan_langsung, df_amr['DAYA (VA)'] / 660 * 1.4)
            
            conditions_pengukuran_langsung = (
                (arus_maks > swither_series)
            ) # otomatis akan False jika Daya tidak ada di dict switcher, jadi tidak perlu filtering daya

            # Pengukuran Tidak Langsung
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            # Kondisi untuk TM
            condition_tm_tidak = (
                (arus_maks > arus_maks_tm) & (~df_amr['DAYA (VA)'].isin(daya_lansung)) & mask_tm
            )
            # Kondisi untuk TR
            condition_tr_tidak = (
               (arus_maks > arus_maks_tr) & (~df_amr['DAYA (VA)'].isin(daya_lansung)) & mask_tr
            )

            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["over_current"] = (
                conditions_pengukuran_langsung | condition_tm_tidak | condition_tr_tidak
            ).fillna(False)

        with co7:
            st.markdown(
                "<h5>Over Voltage</h5>"
                "<p>tegangan berlebih. Tegangan maksimum lebih besar dari ambang yang ditentukan. </p>",
                unsafe_allow_html=True
            )
            st.markdown("------") 
            tegangan_maks_tm = st.number_input(label="Set Tegangan Maksimum (v maks) pada Tegangan Menengah (tm)", value=62.0)
            tegangan_maks_tr = st.number_input(label="Set Tegangan Maksimum (v maks) pada Tegangan Rendah (tr)", value=241.0)

            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            # Kondisi untuk TM
            condition_tm = (
                tegangan_maks > tegangan_maks_tm
            )
            # Kondisi untuk TR
            condition_tr = (
                tegangan_maks > tegangan_maks_tr
            )
            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["over_voltage"] = ((mask_tm & condition_tm) | (mask_tr & condition_tr)).fillna(False)

        st.subheader(" ", divider="green")
        col1, col3, col32, col4, col5 = st.columns((1, 1, 1, 1, 3))
        with col1:
            st.markdown(
                "<h5>Reverse Power</h5>"
                "<p>Active Power kurang dari nilai tertentu <b>namun</b> arus nya melebihi nilai tertentu pada <b>salah satu</b> L1, L2, atau L3.</p>"
                "<p>Selain itu Kami memberikan perbandingan 1:2:4 untuk Ref Billing=2, Ref Billing =1 & siang, Ref Billing =1 & malam</p>",
                unsafe_allow_html=True
            )
            st.markdown("------") 
            active_power_tm = st.number_input(label="Set Non Aktif Power pada Tegangan Menengah | v <", value=0.0)
            active_power_tr = st.number_input(label="Set Non Aktif Power pada Tegangan Rendah | v <", value=0.0)
            batas_bawah_arus_tm_4 = st.number_input(label="Set Batas Bawah Arus tm Reverse Power | I(tm) > ", value=0.5)   
            batas_bawah_arus_tr_4 = st.number_input(label="Set Batas Bawah Arus tr Reverse Power | I(tr) > ", value=0.7)   
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm
            # Kondisi untuk TM
            condition_tm = (
                ((df_amr['ACTIVE_POWER_L1'] < active_power_tm) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tm_4)) |
                ((df_amr['ACTIVE_POWER_L2'] < active_power_tm) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tm_4)) |
                ((df_amr['ACTIVE_POWER_L3'] < active_power_tm) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tm_4))
            )
            # Kondisi untuk TR
            condition_tr = (
                ((df_amr['ACTIVE_POWER_L1'] < active_power_tr) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tr_4)) |
                ((df_amr['ACTIVE_POWER_L2'] < active_power_tr) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tr_4)) |
                ((df_amr['ACTIVE_POWER_L3'] < active_power_tr) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tr_4))
            )
            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["active_power_negative"] = ((mask_tm & condition_tm) | (mask_tr & condition_tr) & (df_amr['BILL_REFF_KWH']==2)).fillna(False)
            df_amr["active_power_negative_siang"] = ((mask_tm & condition_tm) | (mask_tr & condition_tr) & (df_amr['BILL_REFF_KWH']==1) & (df_amr['WAKTU']=="SIANG")).fillna(False)
            df_amr["active_power_negative_malam"] = ((mask_tm & condition_tm) | (mask_tr & condition_tr) & (df_amr['BILL_REFF_KWH']==1) & (df_amr['WAKTU']=="MALAM")).fillna(False)
        with col3:
            st.markdown(
                "<h5>Arus (I) Unbalance</h5>"
                "<p>%Khusus Pengukuran Tak Langsung saja<b>,</b> Deviasi Nilai arus terhadap rata-ratanya >= batas toleransi yang diberikan <b>dan</b> memiliki arus besar pada <b>salah satu</b> L1, L2, atau L3. </p>",
                unsafe_allow_html=True
            )
            st.markdown("------") 
            toleransi_arus_unbalance_tm = st.number_input("Batas Toleransi Arus Unbalance pada Tegangan Menengah", value=0.5, min_value=0.0, max_value=1.0)
            toleransi_arus_unbalance_tr = st.number_input("Batas Toleransi Arus Unbalance pada Tegangan Rendah", value=0.5, min_value=0.0, max_value=1.0)
            batas_bawah_arus_tm_6 = st.number_input(label="Set Batas Bawah Arus tm I unbalance | I(tm) > ", value=0.5)   
            batas_bawah_arus_tr_6 = st.number_input(label="Set Batas Bawah Arus tr I unbalance | I(tr) > ", value=1.0)   
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm
            # Kondisi untuk TM
            rerata_arus = df_amr[['CURRENT_L1', 'CURRENT_L2', 'CURRENT_L3']].mean(axis=1)
            rerata_arus = rerata_arus.replace(0, np.nan)
            condition_tm = (
                ((((df_amr['CURRENT_L1']-rerata_arus).abs()/rerata_arus)>=toleransi_arus_unbalance_tm) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tm_6)) |
                ((((df_amr['CURRENT_L2']-rerata_arus).abs()/rerata_arus)>=toleransi_arus_unbalance_tm) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tm_6)) |
                ((((df_amr['CURRENT_L3']-rerata_arus).abs()/rerata_arus)>=toleransi_arus_unbalance_tm) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tm_6)) 
            )
            # Kondisi untuk TR
            condition_tr = (
                ((((df_amr['CURRENT_L1']-rerata_arus).abs()/rerata_arus)>=toleransi_arus_unbalance_tr) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tr_6)) |
                ((((df_amr['CURRENT_L2']-rerata_arus).abs()/rerata_arus)>=toleransi_arus_unbalance_tr) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tr_6)) |
                ((((df_amr['CURRENT_L3']-rerata_arus).abs()/rerata_arus)>=toleransi_arus_unbalance_tr) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tr_6)) 
            )
            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["unbalance_I"] = (((mask_tm & condition_tm) | (mask_tr & condition_tr)) & (df_amr['Jenis Pengukuran']=="Tak Langsung")).fillna(False)
        with col32:
            st.markdown(
                "<h5>Active Power Lost</h5>"
                "<p>Ada dua syarat: \n(1) Nilai daya/active power bernilai 0 <b>dan</b> arusnya lebih besar dari batas bawah setting-an pada salah satu L1, L2, L3\n(2) Tidak semua nilai daya/active powernya = 0</p>",
                unsafe_allow_html=True
            )
            st.markdown("------") 
            batas_bawah_arus_7 = st.number_input(label="Set Batas Bawah Arus P Lost | I(tm|tr) > ", value=0.5)
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm
            # Kondisi untuk TM | TR (tidak dibagi)
            maks_power = df_amr[['ACTIVE_POWER_L1', 'ACTIVE_POWER_L2', 'ACTIVE_POWER_L3']].max(axis=1)
            condition_apl= (
                ((df_amr['ACTIVE_POWER_L1']==0) & (df_amr['CURRENT_L1'] > batas_bawah_arus_7)) |
                ((df_amr['ACTIVE_POWER_L2']==0) & (df_amr['CURRENT_L2'] > batas_bawah_arus_7)) |
                ((df_amr['ACTIVE_POWER_L3']==0) & (df_amr['CURRENT_L3'] > batas_bawah_arus_7)) 
            )
            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["active_p_lost"] = (condition_apl & (maks_power > 0) & (df_amr['BILL_REFF_KWH']==1)).fillna(False)

        with col4:
            st.markdown(
                "<h5>Arus Lebih Kecil Teg Kecil / Normal</h5>"
                "<p>Kombinasi pasang arus dan tegangan antar L1, L2, dan L3 dengan operator OR.</p>",
                unsafe_allow_html=True
            )
            st.markdown("------") 
            selisih_tegangan_tm = st.number_input("Set Selisih Tegangan pada Tegangan Menenengah (tm)", value=2)
            selisih_tegangan_tr = st.number_input("Set Selisih Tegangan pada Tegangan Rendah (tr)", value=8) 
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm
            # Kondisi untuk TM
            condition_tm = (
                ((df_amr['CURRENT_L1'] < df_amr['CURRENT_L2']) & (df_amr['VOLTAGE_L1'] <= df_amr['VOLTAGE_L2']) & ((df_amr['VOLTAGE_L1']-df_amr['VOLTAGE_L2']).abs()>=selisih_tegangan_tm)) |
                ((df_amr['CURRENT_L1'] < df_amr['CURRENT_L3']) & (df_amr['VOLTAGE_L1'] <= df_amr['VOLTAGE_L3']) & ((df_amr['VOLTAGE_L1']-df_amr['VOLTAGE_L3']).abs()>=selisih_tegangan_tm)) |
                ((df_amr['CURRENT_L2'] < df_amr['CURRENT_L3']) & (df_amr['VOLTAGE_L2'] <= df_amr['VOLTAGE_L3']) & ((df_amr['VOLTAGE_L2']-df_amr['VOLTAGE_L3']).abs()>=selisih_tegangan_tm))
            )
            # Kondisi untuk TR
            condition_tr = (
                ((df_amr['CURRENT_L1'] < df_amr['CURRENT_L2']) & (df_amr['VOLTAGE_L1'] <= df_amr['VOLTAGE_L2']) & ((df_amr['VOLTAGE_L1']-df_amr['VOLTAGE_L2']).abs()>=selisih_tegangan_tr)) |
                ((df_amr['CURRENT_L1'] < df_amr['CURRENT_L3']) & (df_amr['VOLTAGE_L1'] <= df_amr['VOLTAGE_L3']) & ((df_amr['VOLTAGE_L1']-df_amr['VOLTAGE_L3']).abs()>=selisih_tegangan_tr)) |
                ((df_amr['CURRENT_L2'] < df_amr['CURRENT_L3']) & (df_amr['VOLTAGE_L2'] <= df_amr['VOLTAGE_L3']) & ((df_amr['VOLTAGE_L2']-df_amr['VOLTAGE_L3']).abs()>=selisih_tegangan_tr))
            )
            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["arus_kecil_teg_kecil"] = ((mask_tm & condition_tm) | (mask_tr & condition_tr)).fillna(False)
            
            # Freeze 
            jumlah_tegangan = df_amr[['VOLTAGE_L1', 'VOLTAGE_L2', 'VOLTAGE_L3']].sum(axis=1)
            df_amr["freeze"] = (jumlah_tegangan==0).fillna(False)

            # Current Loop
            condition_current_loop = (df_amr['DAYA (VA)'] >= 53000) & (
                (
                    ((df_amr['CURRENT_L1'] - df_amr['CURRENT_L2']).abs() < 0.02) &
                    ((df_amr['CURRENT_L1'] + df_amr['CURRENT_L2']) > 0.6) &
                    ((180 - (df_amr['CURRENT_ANGLE_L1'] - df_amr['CURRENT_ANGLE_L2']).abs()).abs() < 1)
                ) |
                (
                    ((df_amr['CURRENT_L1'] - df_amr['CURRENT_L3']).abs() < 0.02) &
                    ((df_amr['CURRENT_L1'] + df_amr['CURRENT_L3']) > 0.6) &
                    ((180 - (df_amr['CURRENT_ANGLE_L1'] - df_amr['CURRENT_ANGLE_L3']).abs()).abs() < 1)
                ) |
                (
                    ((df_amr['CURRENT_L2'] - df_amr['CURRENT_L3']).abs() < 0.02) &
                    ((df_amr['CURRENT_L2'] + df_amr['CURRENT_L3']) > 0.6) &
                    ((180 - (df_amr['CURRENT_ANGLE_L2'] - df_amr['CURRENT_ANGLE_L3']).abs()).abs() < 1)
                )
            ) 
            df_amr["current_loop"] = condition_current_loop.fillna(False)

        with col5:
            st.subheader('Kriteria TO',divider="red")
            st.write("Untuk menentukan Target Operasi (TO), perlu di tentukan batas minimum kriteria yang dipenuhi, misalnya 2 dari 9 indikator terpenuhi ✅ dan Bobot diatas Nilai Tertentu (Minimal = 2) ✅")
            n_indikator = st.number_input(label="Jumlah Indikator >= ", min_value=1, value=1) 
            # Hitung Potensi TO  
            df_amr['Jumlah Potensi TO'] = df_amr[
                ['v_drop', 'v_lost', 'cos_phi_kecil', 'arus_hilang', 'In_more_Imax',
                'over_current', 'over_voltage', 'active_power_negative', 'active_power_negative_siang', 'active_power_negative_malam', 
                'unbalance_I', 'arus_kecil_teg_kecil', 'current_loop', 'active_p_lost']
            ].astype(int).sum(axis=1)
            # Jumlah terbobot
            df_amr['SUM_WEIGHTED'] = (
                                        condition_v_drop_tak_langsung*20+
                                        condition_v_drop_langsung*7+
                                        df_amr['v_lost']*7+
                                        df_amr['cos_phi_kecil']*10+
                                        df_amr['arus_hilang']*1+
                                        df_amr['In_more_Imax']*10+
                                        df_amr['over_current']*15+
                                        df_amr['over_voltage']*1 +
                                        df_amr['active_power_negative']*1+ 
                                        df_amr['active_power_negative_siang']*7+ 
                                        df_amr['active_power_negative_malam']*10+ 
                                        df_amr['unbalance_I']*3+
                                        df_amr['arus_kecil_teg_kecil']*4+
                                        df_amr['current_loop']*20+
                                        df_amr['active_p_lost']*7+
                                        df_amr['freeze']*20
                                    )
            min_weight, max_weight = min(df_amr['SUM_WEIGHTED']), max(df_amr['SUM_WEIGHTED'])   
            s_weight = st.number_input(label="Jumlah Bobot >= ", min_value=min_weight, value=3, max_value=max_weight)
            n_show = st.number_input(label='Banyak data yang ingin ditampilkan', value=50)  
            
            df_amr = df_amr[(df_amr['Jumlah Potensi TO']>=n_indikator) & (df_amr['SUM_WEIGHTED']>=s_weight) ]
            selected_options = st.multiselect(label='Kriteria Wajib Terpenuhi (Opsional)', options=['v_drop', 'v_lost', 'cos_phi_kecil', 'arus_hilang', 'In_more_Imax',
                'over_current', 'over_voltage', 'active_power_negative', 'active_power_negative_siang', 'active_power_negative_malam',
                'unbalance_I', 'active_p_lost', 'arus_kecil_teg_kecil', 'current_loop', 'freeze'])
            if selected_options:
                df_amr = df_amr[df_amr[selected_options].eq(True).all(axis=1)]
            selected_up = st.multiselect(label="Filter Nama UP (Opsional)", options=df_amr["NAMAUP"].unique())
            if selected_up:
                df_amr = df_amr[df_amr["NAMAUP"].isin(selected_up)]

            st.markdown(
                "<strong>Opsi Fitur Waktu</strong>",
                unsafe_allow_html=True
            )
            # c_pv1, c_pv2 = st.columns(2)
            options_waktu = df_amr['WAKTU'].unique()
            select_waktu = st.multiselect(label="Pilih Waktu Tertentu (Opsional)", options=options_waktu)
            if select_waktu:
                df_amr = df_amr[df_amr['WAKTU'].isin(select_waktu)]
    
    col_dis1, col_dis2, col_dis3 = st.columns(3)
    n_TO = len(df_amr)

    col_dis1.metric(label="Total Data Berhasil di Analisis", value=n_data)
    col_dis2.metric(label="Total IDPEL di Analisis", value=n_idpel)
    col_dis3.metric(label="Target Operasi Memenuhi Kriteria", value=n_TO)

    # Top Rekomendasi
    st.subheader(f"Top {n_show} Rekomendasi Target Operasi Pelanggan AMR Bulan {datetime_data_month} {datetime_data_year}")
    df_amr = df_amr.sort_values(by='SUM_WEIGHTED', ascending=False).reset_index(drop=True)

    df_amr_style = df_amr[['IDPEL','v_drop', 'v_lost', 'cos_phi_kecil', 'arus_hilang', 'In_more_Imax',
                'over_current', 'over_voltage', 'active_power_negative', 'active_power_negative_siang', 'active_power_negative_malam', 
                'unbalance_I', 'active_p_lost', 'arus_kecil_teg_kecil', 'current_loop', 'freeze', 'Jumlah Potensi TO', 'SUM_WEIGHTED']].copy().head(n_show)
    def highlighter(val):
        color = "#D3F6EC" if val == True else 'transparent'
        return f'background-color: {color}'

    df_amr_style = df_amr_style.style.applymap(
        highlighter, 
        subset=['v_drop', 'v_lost', 'cos_phi_kecil', 'arus_hilang', 'In_more_Imax',
                'over_current', 'over_voltage', 'active_power_negative', 'active_power_negative_siang', 'active_power_negative_malam', 
                'unbalance_I', 'active_p_lost', 'arus_kecil_teg_kecil','current_loop', 'freeze']
    )
    st.dataframe(df_amr_style)
    # Tampilkan detail
    # st.write(len(df_amr[(df_amr['IDPEL'].isin(idpel_pv)) & (df_amr["NAMA"].isnull())]))

    st.write("Detail Pelanggan TO")
    st.dataframe(df_amr[["NAMAUP", "IDPEL", "NAMA", "TARIF", "DAYA (VA)", 'Jumlah Potensi TO', 'SUM_WEIGHTED']].head(n_show), use_container_width=True, hide_index=True)
    
    # Buat Download 2 CSV
    # "FAKM"
    SIAP_COL = ["IDPEL", "NAMA", "TARIF", "DAYA (VA)", "Jumlah Potensi TO", "SUM_WEIGHTED"]
    FULL_COL = ["NAMAUP", "IDPEL", "NAMA", "TYPE_METER", "TARIF", "DAYA (VA)", 'WAKTU', 
                "VOLTAGE_L1","VOLTAGE_L2","VOLTAGE_L3", 
                "CURRENT_L1","CURRENT_L2","CURRENT_L3","CURRENT_N",
                'VOLTAGE_ANGLE_L1', 'VOLTAGE_ANGLE_L2', 'VOLTAGE_ANGLE_L3', 
                "CURRENT_ANGLE_L1","CURRENT_ANGLE_L2","CURRENT_ANGLE_L3", 
                "POWER_FACTOR_L1","POWER_FACTOR_L2","POWER_FACTOR_L3","ACTIVE_POWER_L1","ACTIVE_POWER_L2","ACTIVE_POWER_L3","ACTIVE_POWER_TOTAL","KWH_ABS_TOTAL",
                'APPARENT_POWER_L1', 'APPARENT_POWER_L2', 'APPARENT_POWER_L3', 'BILL_REFF_KWH',
                'v_drop', 'v_lost', 'cos_phi_kecil', 'arus_hilang', 'In_more_Imax',
                'over_current', 'over_voltage', 'active_power_negative', 'active_power_negative_siang', 'active_power_negative_malam', 
                'unbalance_I', 'active_p_lost', 'arus_kecil_teg_kecil', 'current_loop', 'freeze', 'Jumlah Potensi TO', "SUM_WEIGHTED"]
    k1, k2, k3, k4 = st.columns(4)
    k1.download_button(
        "Download Data Siap TO",
        df_amr[SIAP_COL].to_csv(index=False, sep=';').encode('utf-8'),
        f"TO_AMR_{datetime_data_month}_{datetime_data_year}.csv",
        "text/csv",
        key='download-csv'
    )

    k2.download_button(
        "Download Data Full Features",
        df_amr[FULL_COL].to_csv(index=False, sep=';').encode('utf-8'),
        f"TO_full_AMR_{datetime_data_month}_{datetime_data_year}.csv",
        "text/csv",
        key='download-full-csv'
    )
# AMR Regular Operate
def operate_amr():
    c2, c1 = st.columns((2, 1))
    c2.markdown("""
    <h3>Otomatisasi Target Operasi <strong>Pelanggan Automatic Meter Reading</strong></h4>
    <p>
    Untuk Menghasilkan Target Operasi pada AMR cukup dibutuhkan 1 data, yaitu: 
    <li>Data Instant AMR, <a href="https://docs.google.com/spreadsheets/d/1jkS97PUU6uUAtOKv94_9-GYpKatLeCXp/edit?usp=drive_link&ouid=113680444123721717153&rtpof=true&sd=true">Contoh Data Instant</a></li>
    </p>
    """, unsafe_allow_html=True)
    uploaded_input_step1  = c1.file_uploader(
        "Upload Data Instant AMR", type=["xlsx", "xls", "csv"]
    )
    if uploaded_input_step1 is None:
        st.info("Silakan unggah file CSV atau Excel untuk memulai.\n\nJika Anda mengalami `AxiosError` saat mengunggah, silakan muat ulang (refresh) halaman. Jika masalah berlanjut, cobalah membersihkan cache browser Anda.")
        return

    placeholderSampleHeader = st.empty()
    placeholderSampleTable = st.empty()
    placeholderSampleSubmit = st.empty()

    INIT_COL_SAMPLE = ["LOCATION_CODE", "LOCATION_TYPE", "TYPE_METER"]
    DATA_TYPES_SAMPLE = {
        "LOCATION_CODE": "string",
        "LOCATION_TYPE": "category",
        "TYPE_METER": "category"
    }

    file_name = uploaded_input_step1.name.lower()
    if file_name.endswith(".csv"):
        (samples, enc), err = errorHandling(lambda:read_csv_try(uploaded_input_step1, 
                         sep=";", nrows=5, usecols=INIT_COL_SAMPLE, dtype=DATA_TYPES_SAMPLE
                          ), "Berhasil Membaca Contoh Data Pelanggan")
    else:
        samples, err = errorHandling(lambda:read_excel(uploaded_input_step1, 
                          nrows=5, usecols=INIT_COL_SAMPLE, dtype=DATA_TYPES_SAMPLE
                          ), "Berhasil Membaca Contoh Data Pelanggan")
    if err is not None:
        return
    
    placeholderSampleHeader.write("Pratinjau Data (5 Baris Pertama)")
    placeholderSampleTable.dataframe(samples, use_container_width=True)

    if not placeholderSampleSubmit.button("Mulai Pemrosesan Data"):
        return
    
    placeholderSampleHeader.empty()
    placeholderSampleTable.empty()
    placeholderSampleSubmit.empty()
    
    INIT_COL = ["NAMAUP", "LOCATION_CODE","NAMA_PELANGGAN", "LOCATION_TYPE", "TYPE_METER", "TARIFF", "POWER", "PERIODE", "READ_DATE","VOLTAGE_L1","VOLTAGE_L2","VOLTAGE_L3","CURRENT_L1","CURRENT_L2","CURRENT_L3","CURRENT_N","ACTIVE_POWER_L1","ACTIVE_POWER_L2","ACTIVE_POWER_L3","ACTIVE_POWER_TOTAL","KWH_ABS_TOTAL","CURRENT_ANGLE_L1","CURRENT_ANGLE_L2","CURRENT_ANGLE_L3", 'VOLTAGE_ANGLE_L1', 'VOLTAGE_ANGLE_L2', 'VOLTAGE_ANGLE_L3', 'APPARENT_POWER_L1', 'APPARENT_POWER_L2', 'APPARENT_POWER_L3', 'BILL_REFF_KWH']
    DATA_TYPES = {
        "LOCATION_CODE": "string",
        "LOCATION_TYPE": "category",
        "TYPE_METER": "category",
        "TARIFF": "category",
        "PERIODE" : "category",
        "POWER" : "category",
        "CURRENT_L1": "float32",
        "CURRENT_L2": "float32",
        "CURRENT_L3": "float32",
        "VOLTAGE_L1": "float32",
        "VOLTAGE_L2": "float32",
        "VOLTAGE_L3": "float32",
        "CURRENT_N": "float32",
        "ACTIVE_POWER_L1": "float32", 
        "ACTIVE_POWER_L2": "float32", 
        "ACTIVE_POWER_L3": "float32", 
        "ACTIVE_POWER_TOTAL": "float32", 
        "KWH_ABS_TOTAL": "float32",
        "CURRENT_ANGLE_L1": "float32",
        "CURRENT_ANGLE_L2": "float32",
        "CURRENT_ANGLE_L3": "float32",
        'VOLTAGE_ANGLE_L1': "float32",
        'VOLTAGE_ANGLE_L2': "float32",
        'VOLTAGE_ANGLE_L3': "float32",
        'APPARENT_POWER_L1': "float32",
        'APPARENT_POWER_L2': "float32",
        'APPARENT_POWER_L3': "float32",
        "BILL_REFF_KWH" : "category"
    }
    parse_dates=["READ_DATE"]

    placeholder1 = st.empty()
    placeholder1.info("⏳ Memuat seluruh data... (Tip: Untuk data berukuran besar, gunakan format CSV dengan delimiter ';' agar jauh lebih cepat)")
    if file_name.endswith(".csv"):
        (df, _), err = errorHandling(lambda:read_csv_try(uploaded_input_step1,
                        sep=";", usecols=INIT_COL, dtype=DATA_TYPES, parse_dates=parse_dates
                        ), "Berhasil Membaca Seluruh Data Pelanggan")
    else:
        df, err = errorHandling(lambda:read_excel(uploaded_input_step1, usecols=INIT_COL, dtype=DATA_TYPES, parse_dates=parse_dates
                          ), "Berhasil Membaca Seluruh Data Pelanggan")
    if err is not None:
        return

    placeholder1.warning("⏳ Data berhasil dimuat. Sedang melakukan analisis target operasi...")
    _, err = errorHandling(lambda:dashboarding_amr(df), "Analisis Target Operasi Selesai")
    if err is not None:
        return
    placeholder1.empty()
    placeholder1.success("🎉 Hasil Analisa sudah keluar !")
    st.divider()

# AMR Worker
def dashboarding_amr(df_amr):
    # "POWER_FACTOR_L1","POWER_FACTOR_L2","POWER_FACTOR_L3"
    df_amr.rename({'LOCATION_CODE': 'IDPEL'}, axis=1, inplace=True)
    df_amr['IDPEL'] = df_amr['IDPEL'].astype(str)
    period_data_y = str(df_amr.loc[0, "PERIODE"])
    period_data = datetime.datetime(int(period_data_y[:4]), int(period_data_y[4:]), 1)
    datetime_data_month = period_data.strftime("%B")
    datetime_data_year = period_data.strftime("%Y")
    
    st.markdown(f"""<h2>Dashboard Target Operasi P2TL AMR Periode {datetime_data_month} {datetime_data_year}</h2>""", unsafe_allow_html=True)
    df_amr['JAM'] = df_amr['READ_DATE'].dt.hour
    df_amr['WAKTU'] = np.where((df_amr['JAM'] < 6) | (df_amr['JAM'] > 18), "MALAM", "SIANG")
    df_amr = df_amr[df_amr['LOCATION_TYPE']=="CUSTOMER"] # Filter CUSTOMER
    # Handle duplicate
    df_amr = df_amr.sort_values(by='READ_DATE', ascending=False).drop_duplicates(subset=['IDPEL', 'WAKTU']).reset_index(drop=True)
    n_data = len(df_amr)
    n_idpel = len(df_amr['IDPEL'].unique())

    tegangan_maks = df_amr[['VOLTAGE_L1', 'VOLTAGE_L2', 'VOLTAGE_L3']].max(axis=1).astype(float)
    df_amr['PENG'] = np.where(tegangan_maks> 70, "TR", "TM")

    try: 
        df_amr['POWER'] = df_amr['POWER'].astype(int)
    except Exception as e:
        df_amr['POWER'] = df_amr['POWER'].str.replace(',', '').astype(int)
    df_amr.rename({'NAMA_PELANGGAN':'NAMA', 'TARIFF':'TARIF', 'POWER':'DAYA (VA)'}, axis=1, inplace=True)

    # Jenis Pengukuran
    df_amr['Jenis Pengukuran'] = np.where(df_amr['DAYA (VA)'] < 53000, "Langsung", "Tak Langsung")
    df_amr['PHASE'] = np.where(
        (df_amr['DAYA (VA)'] > 11000) | (df_amr['DAYA (VA)'] == 10600) | (df_amr['DAYA (VA)'] == 6600),
        3,1)
    df_amr['BILL_REFF_KWH'] = df_amr['BILL_REFF_KWH'].astype(int)

    # benerin to float 
    float_column = [
        'VOLTAGE_L1', 'VOLTAGE_L2', 'VOLTAGE_L3',
        'VOLTAGE_ANGLE_L1', 'VOLTAGE_ANGLE_L2', 'VOLTAGE_ANGLE_L3', 
        'CURRENT_L1', 'CURRENT_L2', 'CURRENT_L3',
        'CURRENT_N', 'ACTIVE_POWER_L1', 'ACTIVE_POWER_L2',
        'ACTIVE_POWER_L3', 'ACTIVE_POWER_TOTAL', 'KWH_ABS_TOTAL',
        'CURRENT_ANGLE_L1', 'CURRENT_ANGLE_L2', 'CURRENT_ANGLE_L3', 
        'APPARENT_POWER_L1', 'APPARENT_POWER_L2', 'APPARENT_POWER_L3'
    ]
    for col in float_column:
        df_amr[col] = pd.to_numeric(
            df_amr[col].astype(str).str.replace(',', '.', regex=False),
            errors='coerce'
        )

    # COS PHI MEASUREMENT
    df_amr["POWER_FACTOR_L1"] = np.where(
        df_amr["APPARENT_POWER_L1"] != 0,
        df_amr["ACTIVE_POWER_L1"] / df_amr["APPARENT_POWER_L1"],
        np.cos(np.radians(df_amr["CURRENT_ANGLE_L1"] - df_amr["VOLTAGE_ANGLE_L1"]))
    )
    df_amr["POWER_FACTOR_L2"] = np.where(
        df_amr["APPARENT_POWER_L2"] != 0,
        df_amr["ACTIVE_POWER_L2"] / df_amr["APPARENT_POWER_L2"],
        np.cos(np.radians(df_amr["CURRENT_ANGLE_L2"] - df_amr["VOLTAGE_ANGLE_L2"]))
    )
    df_amr["POWER_FACTOR_L3"] = np.where(
        df_amr["APPARENT_POWER_L3"] != 0,
        df_amr["ACTIVE_POWER_L3"] / df_amr["APPARENT_POWER_L3"],
        np.cos(np.radians(df_amr["CURRENT_ANGLE_L3"] - df_amr["VOLTAGE_ANGLE_L3"]))
    )

    with st.expander("Setting Parameter 📐"):
        st.write("Operasi Logika yang digunakan disini adalah OR. Dengan demikian, indikator yang sesuai dengan salah satu spesifikasi aturan tersebut akan di highlight berwarna hijau cerah dan berkontribusi pada perhitungan potensi TO.")
        # FILTER
        co1, co2, co3, co4, co5, co6, co7 = st.columns(7)
        with co1: 
            st.markdown(
                "<h5>Tegangan Drop</h5>"
                "<p>Tegangan drop secara sederhana dapat terjadi ketika <b>salah satu</b> L1, L2, L3 ber tegangan kecil <b>dan</b> bernilai positif <b>dan</b> ber arus besar.</p>",
                unsafe_allow_html=True
            )
            st.markdown("------")
            batas_atas_v_tm = st.number_input(label="Set Batas Atas Tegangan Menengah (tm) | tm < ", value=56.0)        
            batas_atas_v_tr = st.number_input(label="Set Batas Atas Tegangan Rendah (tr) | tr < ", value=180.0)                
            batas_bawah_arus_tm = st.number_input(label="Set Batas Bawah Arus Besar tm | I(tm) > ", value=0.5)   
            batas_bawah_arus_tr = st.number_input(label="Set Batas Bawah Arus Besar tr | I(tr) > ", value=0.5)   
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            # Kondisi untuk TM
            condition_tm = (
                ((df_amr['VOLTAGE_L1'] < batas_atas_v_tm) & (df_amr['VOLTAGE_L1'] >= 0) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tm)) |
                ((df_amr['VOLTAGE_L2'] < batas_atas_v_tm) & (df_amr['VOLTAGE_L2'] >= 0) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tm)) |
                ((df_amr['VOLTAGE_L3'] < batas_atas_v_tm) & (df_amr['VOLTAGE_L3'] >= 0) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tm))
            )
            # Kondisi untuk TR
            condition_tr = (
                ((df_amr['VOLTAGE_L1'] < batas_atas_v_tr) & (df_amr['VOLTAGE_L1'] >= 0) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tr)) |
                ((df_amr['VOLTAGE_L2'] < batas_atas_v_tr) & (df_amr['VOLTAGE_L2'] >= 0) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tr)) |
                ((df_amr['VOLTAGE_L3'] < batas_atas_v_tr) & (df_amr['VOLTAGE_L3'] >= 0) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tr))
            )

            # Gabungkan kondisi berdasarkan tipe PENG
            condition_v_drop = (mask_tm & condition_tm) | (mask_tr & condition_tr)
            df_amr["v_drop"] = condition_v_drop
            condition_v_drop_tak_langsung = condition_v_drop & (df_amr['DAYA (VA)'] >= 53000)
            condition_v_drop_langsung = condition_v_drop & (df_amr['DAYA (VA)'] < 53000)
        
        with co2:
            st.markdown(
                "<h5>Tegangan Hilang</h5>"
                "<p>Tegangan hilang adalah data dengan PHASE 3 yang tegangannya hilang <b>dan</b> arus (I) lebih dari ambang <b>pada salah satu</b> L1, L2, atau L3. </p>",
                unsafe_allow_html=True
            )
            st.markdown("------")
            tegangan_hilang_tm = st.number_input(label="Nilai Tegangan Menengah Hilang (tm) | tm = ", value=0.0)
            tegangan_hilang_tr = st.number_input(label="Nilai Tegangan Rendah Hilang (tr) | tr = ", value=0.0)
            batas_bawah_arus_tm_2 = st.number_input(label="Set Batas Bawah Arus Besar tm | I(tm) > ", value=-1.0)   
            batas_bawah_arus_tr_2 = st.number_input(label="Set Batas Bawah Arus Besar tr | I(tr) > ", value=-1.0)   
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            # Kondisi untuk TM
            condition_tm = (
                (df_amr['PHASE']==3) & (
                    ((df_amr['VOLTAGE_L1'] == tegangan_hilang_tm) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tm_2)) |
                    ((df_amr['VOLTAGE_L2'] == tegangan_hilang_tm) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tm_2)) |
                    ((df_amr['VOLTAGE_L3'] == tegangan_hilang_tm) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tm_2))
                )
            )
            # Kondisi untuk TR
            condition_tr = (
                (df_amr['PHASE']==3) & (
                    ((df_amr['VOLTAGE_L1'] == tegangan_hilang_tr) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tr_2)) |
                    ((df_amr['VOLTAGE_L2'] == tegangan_hilang_tr) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tr_2)) |
                    ((df_amr['VOLTAGE_L3'] == tegangan_hilang_tr) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tr_2))
                )
            )
            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["v_lost"] = ((mask_tm & condition_tm) | (mask_tr & condition_tr)).fillna(False)

        with co3: 
            st.markdown(
                "<h5>Cos Phi Kecil</h5>"
                "<p>Cos phi kecil atau besaran daya yang dipakai untuk kerja kecil <b>dan</b> pada salah satu L1, L2, atau L3 <b>dan</b> khusus tak langsung.</p>",
                unsafe_allow_html=True
            )
            st.markdown("------")
            batas_atas_cos_phi_tm = st.number_input(label="Set Batas Atas Cos Phi Kecil pada Tegangan Menengah (tm) | cos phi <= ", value=0.4)
            batas_atas_cos_phi_tr = st.number_input(label="Set Batas Atas Cos Phi Kecil pada Tegangan Rendah (tr) | cos phi <= ", value=0.4)
            batas_bawah_arus_tm_3 = st.number_input(label="Set Batas Bawah Arus Besar tm cos phi kecil | I(tm) > ", value=0.8)   
            batas_bawah_arus_tr_3 = st.number_input(label="Set Batas Bawah Arus Besar tr cos phi kecil | I(tr) > ", value=0.8)   
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            # Kondisi untuk TM
            condition_tm = (
                ((df_amr['POWER_FACTOR_L1'].abs() <= batas_atas_cos_phi_tm) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tm_3)) |
                ((df_amr['POWER_FACTOR_L2'].abs() <= batas_atas_cos_phi_tm) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tm_3)) |
                ((df_amr['POWER_FACTOR_L3'].abs() <= batas_atas_cos_phi_tm) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tm_3))
            )
            # Kondisi untuk TR
            condition_tr = (
                ((df_amr['POWER_FACTOR_L1'].abs() <= batas_atas_cos_phi_tr) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tr_3)) |
                ((df_amr['POWER_FACTOR_L2'].abs() <= batas_atas_cos_phi_tr) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tr_3)) |
                ((df_amr['POWER_FACTOR_L3'].abs() <= batas_atas_cos_phi_tr) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tr_3))
            )

            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["cos_phi_kecil"] = (((mask_tm & condition_tm) | (mask_tr & condition_tr)) & (df_amr['Jenis Pengukuran']=="Tak Langsung")).fillna(False)

        with co4: 
            st.markdown(
                "<h5>Arus Hilang</h5>"
                "<p>arus hilang terjadi jika arus nya kecil <b>dan</b> arusnya pernah besar <b>pada salah satu</b> L1, L2, atau L3. </p>",
                unsafe_allow_html=True
            )
            st.markdown("------")  
            batas_atas_arus_hilang_tm = st.number_input(label="Set Batas Atas arus hilang pada Tegangan Menengah (tm) | arus <= ", value=0.02)
            batas_atas_arus_hilang_tr = st.number_input(label="Set Batas Atas arus hilang pada Tegangan Rendah (tr) | arus <= ", value=0.02)
            batas_bawah_arus_maks_tm= st.number_input(label="Set Batas Bawah Arus Maksimum tm | I maks > ", value=1.0)   
            batas_bawah_arus_maks_tr = st.number_input(label="Set Batas Bawah Arus Maksimum tr | I maks >", value=1.0)   
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            arus_maks = df_amr[['CURRENT_L1', 'CURRENT_L2', 'CURRENT_L3']].max(axis=1)
            arus_mins = df_amr[['CURRENT_L1', 'CURRENT_L2', 'CURRENT_L3']].min(axis=1)
            # Kondisi untuk TM
            condition_tm = (
                ((df_amr['CURRENT_L1'] <= batas_atas_arus_hilang_tm) & (arus_maks > batas_bawah_arus_maks_tm)) |
                ((df_amr['CURRENT_L2'] <= batas_atas_arus_hilang_tm) & (arus_maks > batas_bawah_arus_maks_tm)) |
                ((df_amr['CURRENT_L3'] <= batas_atas_arus_hilang_tm) & (arus_maks > batas_bawah_arus_maks_tm))
            )
            # Kondisi untuk TR
            condition_tr = (
                ((df_amr['CURRENT_L1'] <= batas_atas_arus_hilang_tr) & (arus_maks > batas_bawah_arus_maks_tr)) |
                ((df_amr['CURRENT_L2'] <= batas_atas_arus_hilang_tr) & (arus_maks > batas_bawah_arus_maks_tr)) |
                ((df_amr['CURRENT_L3'] <= batas_atas_arus_hilang_tr) & (arus_maks > batas_bawah_arus_maks_tr))
            )

            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["arus_hilang"] = ((mask_tm & condition_tm) | (mask_tr & condition_tr)).fillna(False)

        with co5:
            st.markdown(
                "<h5>Arus Netral Lebih besar dari Arus Maksimum</h5>"
                "<p>kriterianya adalah arus netral lebih besar dari (arus maksimum - arus minimum + 5) <b>dan</b> lebih besar dari nilai arus netral yang ditentukan.</p>",
                unsafe_allow_html=True
            )
            st.warning("⚠️ Untuk Tipe Meter HXE310, AMETER300, MK10MI harap dilakukan evaluasi lebih lanjut")
            st.markdown("------")  
            batas_bawah_arus_netral_tm = st.number_input("Set Batas Bawah Arus Netral tm | I netral > ", value=1.0)
            batas_bawah_arus_netral_tr = st.number_input("Set Batas Bawah Arus Netral tr | I netral > ", value=10.0)
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            # Kondisi untuk TM
            condition_tm = (
                ((df_amr['CURRENT_N'] > (arus_maks-arus_mins+5)) & (df_amr['CURRENT_N'] > batas_bawah_arus_netral_tm))
            )
            # Kondisi untuk TR
            condition_tr = (
                ((df_amr['CURRENT_N'] > (arus_maks-arus_mins+5)) & (df_amr['CURRENT_N'] > batas_bawah_arus_netral_tr))
            )

            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["In_more_Imax"] = ((mask_tm & condition_tm) | (mask_tr & condition_tr)).fillna(False)

        with co6:
            # Logika di SHEET untuk kolom AH sepertinya masih kurang 
            st.markdown(
                "<h5>Over Current</h5>"
                "<p>Nilai arus Maks(L1, L2, L3) lebih besar dari nilai yang kami tentukan (untuk daya tertentu) + Nilai Arus Maks(L1, L2, L3) yang lebih besar dari nilai yang ditentukan user. </p>",
                unsafe_allow_html=True
            )
            st.markdown("------") 
            arus_maks_tm = st.number_input(label="Set Batas bawah Arus Maks (Pengukuran Tak Langsung) pada Tegangan Menengah (tm)", value=5.0)
            arus_maks_tr = st.number_input(label="Set Batas bawah Arus Maks (Pengukuran Tak Langsung) pada Tegangan Rendah (tr)", value=5.0)

            # pengukuran langsung
            daya_lansung = [450, 900, 1300, 2200, 3500, 4400, 5500, 6600, 7700, 10600, 11000, 13200, 16500, 23000, 33000, 41500]
            swither_over_current = {
                450: 2, 
                900: 4, 
                1300: 6, 
                2200: 10, 
                3500: 16, 
                4400: 20, 
                5500: 25, 
                6600: 10, 
                7700: 35, 
                10600: 16, 
                11000: 50, 
                13200: 20, 
                16500: 25, 
                23000: 35, 
                33000: 50, 
                41500: 63
            }
            swither_series = df_amr['DAYA (VA)'].map(swither_over_current)*1.4  # konversi daya → nilai arus ambang
            # FIX: vectorized fill — lebih cepat dari Python loop untuk data besar
            mask_nan_langsung = swither_series.isna() & (df_amr['DAYA (VA)'] < 53000)
            swither_series = swither_series.where(~mask_nan_langsung, df_amr['DAYA (VA)'] / 660 * 1.4)
            
            conditions_pengukuran_langsung = (
                (arus_maks > swither_series)
            ) # otomatis akan False jika Daya tidak ada di dict switcher, jadi tidak perlu filtering daya

            # Pengukuran Tidak Langsung
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            # Kondisi untuk TM
            condition_tm_tidak = (
                (arus_maks > arus_maks_tm) & (~df_amr['DAYA (VA)'].isin(daya_lansung)) & mask_tm
            )
            # Kondisi untuk TR
            condition_tr_tidak = (
               (arus_maks > arus_maks_tr) & (~df_amr['DAYA (VA)'].isin(daya_lansung)) & mask_tr
            )

            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["over_current"] = (
                conditions_pengukuran_langsung | condition_tm_tidak | condition_tr_tidak
            ).fillna(False)

        with co7:
            st.markdown(
                "<h5>Over Voltage</h5>"
                "<p>tegangan berlebih. Tegangan maksimum lebih besar dari ambang yang ditentukan. </p>",
                unsafe_allow_html=True
            )
            st.markdown("------") 
            tegangan_maks_tm = st.number_input(label="Set Tegangan Maksimum (v maks) pada Tegangan Menengah (tm)", value=62.0)
            tegangan_maks_tr = st.number_input(label="Set Tegangan Maksimum (v maks) pada Tegangan Rendah (tr)", value=241.0)

            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm

            # Kondisi untuk TM
            condition_tm = (
                tegangan_maks > tegangan_maks_tm
            )
            # Kondisi untuk TR
            condition_tr = (
                tegangan_maks > tegangan_maks_tr
            )
            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["over_voltage"] = ((mask_tm & condition_tm) | (mask_tr & condition_tr)).fillna(False)

        st.subheader(" ", divider="green")
        col1, col3, col32, col4, col5 = st.columns((1, 1, 1, 1, 3))
        with col1:
            st.markdown(
                "<h5>Reverse Power</h5>"
                "<p>Active Power kurang dari nilai tertentu <b>namun</b> arus nya melebihi nilai tertentu pada <b>salah satu</b> L1, L2, atau L3.</p>"
                "<p>Selain itu Kami memberikan perbandingan 1:2:4 untuk Ref Billing=2, Ref Billing =1 & siang, Ref Billing =1 & malam</p>",
                unsafe_allow_html=True
            )
            st.markdown("------") 
            active_power_tm = st.number_input(label="Set Non Aktif Power pada Tegangan Menengah | v <", value=0.0)
            active_power_tr = st.number_input(label="Set Non Aktif Power pada Tegangan Rendah | v <", value=0.0)
            batas_bawah_arus_tm_4 = st.number_input(label="Set Batas Bawah Arus tm Reverse Power | I(tm) > ", value=0.5)   
            batas_bawah_arus_tr_4 = st.number_input(label="Set Batas Bawah Arus tr Reverse Power | I(tr) > ", value=0.7)   
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm
            # Kondisi untuk TM
            condition_tm = (
                ((df_amr['ACTIVE_POWER_L1'] < active_power_tm) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tm_4)) |
                ((df_amr['ACTIVE_POWER_L2'] < active_power_tm) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tm_4)) |
                ((df_amr['ACTIVE_POWER_L3'] < active_power_tm) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tm_4))
            )
            # Kondisi untuk TR
            condition_tr = (
                ((df_amr['ACTIVE_POWER_L1'] < active_power_tr) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tr_4)) |
                ((df_amr['ACTIVE_POWER_L2'] < active_power_tr) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tr_4)) |
                ((df_amr['ACTIVE_POWER_L3'] < active_power_tr) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tr_4))
            )
            # Gabungkan kondisi berdasarkan tipe PENG
            # FIX: tambah kurung luar agar filter BILL_REFF_KWH berlaku untuk BOTH TM & TR
            df_amr["active_power_negative"] = (((mask_tm & condition_tm) | (mask_tr & condition_tr)) & (df_amr['BILL_REFF_KWH']==2)).fillna(False)
            df_amr["active_power_negative_siang"] = (((mask_tm & condition_tm) | (mask_tr & condition_tr)) & (df_amr['BILL_REFF_KWH']==1) & (df_amr['WAKTU']=="SIANG")).fillna(False)
            df_amr["active_power_negative_malam"] = (((mask_tm & condition_tm) | (mask_tr & condition_tr)) & (df_amr['BILL_REFF_KWH']==1) & (df_amr['WAKTU']=="MALAM")).fillna(False)
        with col3:
            st.markdown(
                "<h5>Arus (I) Unbalance</h5>"
                "<p>%Khusus Pengukuran Tak Langsung saja<b>,</b> Deviasi Nilai arus terhadap rata-ratanya >= batas toleransi yang diberikan <b>dan</b> memiliki arus besar pada <b>salah satu</b> L1, L2, atau L3. </p>",
                unsafe_allow_html=True
            )
            st.markdown("------") 
            toleransi_arus_unbalance_tm = st.number_input("Batas Toleransi Arus Unbalance pada Tegangan Menengah", value=0.5, min_value=0.0, max_value=1.0)
            toleransi_arus_unbalance_tr = st.number_input("Batas Toleransi Arus Unbalance pada Tegangan Rendah", value=0.5, min_value=0.0, max_value=1.0)
            batas_bawah_arus_tm_6 = st.number_input(label="Set Batas Bawah Arus tm I unbalance | I(tm) > ", value=0.5)   
            batas_bawah_arus_tr_6 = st.number_input(label="Set Batas Bawah Arus tr I unbalance | I(tr) > ", value=1.0)   
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm
            # Kondisi untuk TM
            rerata_arus = df_amr[['CURRENT_L1', 'CURRENT_L2', 'CURRENT_L3']].mean(axis=1)
            rerata_arus = rerata_arus.replace(0, np.nan)
            condition_tm = (
                ((((df_amr['CURRENT_L1']-rerata_arus).abs()/rerata_arus)>=toleransi_arus_unbalance_tm) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tm_6)) |
                ((((df_amr['CURRENT_L2']-rerata_arus).abs()/rerata_arus)>=toleransi_arus_unbalance_tm) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tm_6)) |
                ((((df_amr['CURRENT_L3']-rerata_arus).abs()/rerata_arus)>=toleransi_arus_unbalance_tm) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tm_6)) 
            )
            # Kondisi untuk TR
            condition_tr = (
                ((((df_amr['CURRENT_L1']-rerata_arus).abs()/rerata_arus)>=toleransi_arus_unbalance_tr) & (df_amr['CURRENT_L1'] > batas_bawah_arus_tr_6)) |
                ((((df_amr['CURRENT_L2']-rerata_arus).abs()/rerata_arus)>=toleransi_arus_unbalance_tr) & (df_amr['CURRENT_L2'] > batas_bawah_arus_tr_6)) |
                ((((df_amr['CURRENT_L3']-rerata_arus).abs()/rerata_arus)>=toleransi_arus_unbalance_tr) & (df_amr['CURRENT_L3'] > batas_bawah_arus_tr_6)) 
            )
            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["unbalance_I"] = (((mask_tm & condition_tm) | (mask_tr & condition_tr)) & (df_amr['Jenis Pengukuran']=="Tak Langsung")).fillna(False)
        with col32:
            st.markdown(
                "<h5>Active Power Lost</h5>"
                "<p>Ada dua syarat: \n(1) Nilai daya/active power bernilai 0 <b>dan</b> arusnya lebih besar dari batas bawah setting-an pada salah satu L1, L2, L3\n(2) Tidak semua nilai daya/active powernya = 0</p>",
                unsafe_allow_html=True
            )
            st.markdown("------") 
            batas_bawah_arus_7 = st.number_input(label="Set Batas Bawah Arus P Lost | I(tm|tr) > ", value=0.5)
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm
            # Kondisi untuk TM | TR (tidak dibagi)
            maks_power = df_amr[['ACTIVE_POWER_L1', 'ACTIVE_POWER_L2', 'ACTIVE_POWER_L3']].max(axis=1)
            condition_apl= (
                ((df_amr['ACTIVE_POWER_L1']==0) & (df_amr['CURRENT_L1'] > batas_bawah_arus_7)) |
                ((df_amr['ACTIVE_POWER_L2']==0) & (df_amr['CURRENT_L2'] > batas_bawah_arus_7)) |
                ((df_amr['ACTIVE_POWER_L3']==0) & (df_amr['CURRENT_L3'] > batas_bawah_arus_7)) 
            )
            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["active_p_lost"] = (condition_apl & (maks_power > 0) & (df_amr['BILL_REFF_KWH']==1)).fillna(False)

        with col4:
            st.markdown(
                "<h5>Arus Lebih Kecil Teg Kecil / Normal</h5>"
                "<p>Kombinasi pasang arus dan tegangan antar L1, L2, dan L3 dengan operator OR.</p>",
                unsafe_allow_html=True
            )
            st.markdown("------") 
            selisih_tegangan_tm = st.number_input("Set Selisih Tegangan pada Tegangan Menenengah (tm)", value=2)
            selisih_tegangan_tr = st.number_input("Set Selisih Tegangan pada Tegangan Rendah (tr)", value=8) 
            mask_tm = (df_amr['PENG'] == "TM")
            mask_tr = ~mask_tm
            # Kondisi untuk TM
            condition_tm = (
                ((df_amr['CURRENT_L1'] < df_amr['CURRENT_L2']) & (df_amr['VOLTAGE_L1'] <= df_amr['VOLTAGE_L2']) & ((df_amr['VOLTAGE_L1']-df_amr['VOLTAGE_L2']).abs()>=selisih_tegangan_tm)) |
                ((df_amr['CURRENT_L1'] < df_amr['CURRENT_L3']) & (df_amr['VOLTAGE_L1'] <= df_amr['VOLTAGE_L3']) & ((df_amr['VOLTAGE_L1']-df_amr['VOLTAGE_L3']).abs()>=selisih_tegangan_tm)) |
                ((df_amr['CURRENT_L2'] < df_amr['CURRENT_L3']) & (df_amr['VOLTAGE_L2'] <= df_amr['VOLTAGE_L3']) & ((df_amr['VOLTAGE_L2']-df_amr['VOLTAGE_L3']).abs()>=selisih_tegangan_tm))
            )
            # Kondisi untuk TR
            condition_tr = (
                ((df_amr['CURRENT_L1'] < df_amr['CURRENT_L2']) & (df_amr['VOLTAGE_L1'] <= df_amr['VOLTAGE_L2']) & ((df_amr['VOLTAGE_L1']-df_amr['VOLTAGE_L2']).abs()>=selisih_tegangan_tr)) |
                ((df_amr['CURRENT_L1'] < df_amr['CURRENT_L3']) & (df_amr['VOLTAGE_L1'] <= df_amr['VOLTAGE_L3']) & ((df_amr['VOLTAGE_L1']-df_amr['VOLTAGE_L3']).abs()>=selisih_tegangan_tr)) |
                ((df_amr['CURRENT_L2'] < df_amr['CURRENT_L3']) & (df_amr['VOLTAGE_L2'] <= df_amr['VOLTAGE_L3']) & ((df_amr['VOLTAGE_L2']-df_amr['VOLTAGE_L3']).abs()>=selisih_tegangan_tr))
            )
            # Gabungkan kondisi berdasarkan tipe PENG
            df_amr["arus_kecil_teg_kecil"] = ((mask_tm & condition_tm) | (mask_tr & condition_tr)).fillna(False)
            
            # Freeze 
            jumlah_tegangan = df_amr[['VOLTAGE_L1', 'VOLTAGE_L2', 'VOLTAGE_L3']].sum(axis=1)
            df_amr["freeze"] = (jumlah_tegangan==0).fillna(False)

            # Current Loop
            condition_current_loop = (df_amr['DAYA (VA)'] >= 53000) & (
                (
                    ((df_amr['CURRENT_L1'] - df_amr['CURRENT_L2']).abs() < 0.02) &
                    ((df_amr['CURRENT_L1'] + df_amr['CURRENT_L2']) > 0.6) &
                    ((180 - (df_amr['CURRENT_ANGLE_L1'] - df_amr['CURRENT_ANGLE_L2']).abs()).abs() < 1)
                ) |
                (
                    ((df_amr['CURRENT_L1'] - df_amr['CURRENT_L3']).abs() < 0.02) &
                    ((df_amr['CURRENT_L1'] + df_amr['CURRENT_L3']) > 0.6) &
                    ((180 - (df_amr['CURRENT_ANGLE_L1'] - df_amr['CURRENT_ANGLE_L3']).abs()).abs() < 1)
                ) |
                (
                    ((df_amr['CURRENT_L2'] - df_amr['CURRENT_L3']).abs() < 0.02) &
                    ((df_amr['CURRENT_L2'] + df_amr['CURRENT_L3']) > 0.6) &
                    ((180 - (df_amr['CURRENT_ANGLE_L2'] - df_amr['CURRENT_ANGLE_L3']).abs()).abs() < 1)
                )
            ) 
            df_amr["current_loop"] = condition_current_loop.fillna(False)

        with col5:
            st.subheader('Kriteria TO',divider="red")
            st.write("Untuk menentukan Target Operasi (TO), perlu di tentukan batas minimum kriteria yang dipenuhi, misalnya 2 dari 9 indikator terpenuhi ✅ dan Bobot diatas Nilai Tertentu (Minimal = 2) ✅")
            n_indikator = st.number_input(label="Jumlah Indikator >= ", min_value=1, value=1) 
            # Hitung Potensi TO  
            df_amr['Jumlah Potensi TO'] = df_amr[
                ['v_drop', 'v_lost', 'cos_phi_kecil', 'arus_hilang', 'In_more_Imax',
                'over_current', 'over_voltage', 'active_power_negative', 'active_power_negative_siang', 'active_power_negative_malam', 
                'unbalance_I', 'arus_kecil_teg_kecil', 'current_loop', 'active_p_lost']
            ].astype(int).sum(axis=1)
            # Jumlah terbobot
            df_amr['SUM_WEIGHTED'] = (
                                        condition_v_drop_tak_langsung*20+
                                        condition_v_drop_langsung*7+
                                        df_amr['v_lost']*7+
                                        df_amr['cos_phi_kecil']*10+
                                        df_amr['arus_hilang']*1+
                                        df_amr['In_more_Imax']*10+
                                        df_amr['over_current']*15+
                                        df_amr['over_voltage']*1 +
                                        df_amr['active_power_negative']*1+ 
                                        df_amr['active_power_negative_siang']*7+ 
                                        df_amr['active_power_negative_malam']*10+ 
                                        df_amr['unbalance_I']*3+
                                        df_amr['arus_kecil_teg_kecil']*4+
                                        df_amr['current_loop']*20+
                                        df_amr['active_p_lost']*7+
                                        df_amr['freeze']*20
                                    )
            min_weight, max_weight = min(df_amr['SUM_WEIGHTED']), max(df_amr['SUM_WEIGHTED'])   
            s_weight = st.number_input(label="Jumlah Bobot >= ", min_value=min_weight, value=3, max_value=max_weight)
            n_show = st.number_input(label='Banyak data yang ingin ditampilkan', value=50)  
            
            df_amr = df_amr[(df_amr['Jumlah Potensi TO']>=n_indikator) & (df_amr['SUM_WEIGHTED']>=s_weight) ]
            selected_options = st.multiselect(label='Kriteria Wajib Terpenuhi (Opsional)', options=['v_drop', 'v_lost', 'cos_phi_kecil', 'arus_hilang', 'In_more_Imax',
                'over_current', 'over_voltage', 'active_power_negative', 'active_power_negative_siang', 'active_power_negative_malam',
                'unbalance_I', 'active_p_lost', 'arus_kecil_teg_kecil', 'current_loop', 'freeze'])
            if selected_options:
                df_amr = df_amr[df_amr[selected_options].eq(True).all(axis=1)]
            selected_up = st.multiselect(label="Filter Nama UP (Opsional)", options=df_amr["NAMAUP"].unique())
            if selected_up:
                df_amr = df_amr[df_amr["NAMAUP"].isin(selected_up)]

            st.markdown(
                "<strong>Opsi Fitur Waktu</strong>",
                unsafe_allow_html=True
            )
            # c_pv1, c_pv2 = st.columns(2)
            options_waktu = df_amr['WAKTU'].unique()
            select_waktu = st.multiselect(label="Pilih Waktu Tertentu (Opsional)", options=options_waktu)
            if select_waktu:
                df_amr = df_amr[df_amr['WAKTU'].isin(select_waktu)]
    
    col_dis1, col_dis2, col_dis3 = st.columns(3)
    n_TO = len(df_amr)

    col_dis1.metric(label="Total Data Berhasil di Analisis", value=n_data)
    col_dis2.metric(label="Total IDPEL di Analisis", value=n_idpel)
    col_dis3.metric(label="Target Operasi Memenuhi Kriteria", value=n_TO)

    # Top Rekomendasi
    st.subheader(f"Top {n_show} Rekomendasi Target Operasi Pelanggan AMR Bulan {datetime_data_month} {datetime_data_year}")
    df_amr = df_amr.sort_values(by='SUM_WEIGHTED', ascending=False).reset_index(drop=True)

    df_amr_style = df_amr[['IDPEL','v_drop', 'v_lost', 'cos_phi_kecil', 'arus_hilang', 'In_more_Imax',
                'over_current', 'over_voltage', 'active_power_negative', 'active_power_negative_siang', 'active_power_negative_malam', 
                'unbalance_I', 'active_p_lost', 'arus_kecil_teg_kecil', 'current_loop', 'freeze', 'Jumlah Potensi TO', 'SUM_WEIGHTED']].copy().head(n_show)

    df_amr_style = df_amr_style.style.applymap(
        highlighter, 
        subset=['v_drop', 'v_lost', 'cos_phi_kecil', 'arus_hilang', 'In_more_Imax',
                'over_current', 'over_voltage', 'active_power_negative', 'active_power_negative_siang', 'active_power_negative_malam', 
                'unbalance_I', 'active_p_lost', 'arus_kecil_teg_kecil','current_loop', 'freeze']
    )
    st.dataframe(df_amr_style)
    # Tampilkan detail
    # st.write(len(df_amr[(df_amr['IDPEL'].isin(idpel_pv)) & (df_amr["NAMA"].isnull())]))

    st.write("Detail Pelanggan TO")
    st.dataframe(df_amr[["NAMAUP", "IDPEL", "NAMA", "TARIF", "DAYA (VA)", 'Jumlah Potensi TO', 'SUM_WEIGHTED']].head(n_show), use_container_width=True, hide_index=True)
    
    # Buat Download 2 CSV
    # "FAKM"
    SIAP_COL = ["IDPEL", "NAMA", "TARIF", "DAYA (VA)", "Jumlah Potensi TO", "SUM_WEIGHTED"]
    FULL_COL = ["NAMAUP", "IDPEL", "NAMA", "TYPE_METER", "TARIF", "DAYA (VA)", 'WAKTU', 
                "VOLTAGE_L1","VOLTAGE_L2","VOLTAGE_L3", 
                "CURRENT_L1","CURRENT_L2","CURRENT_L3","CURRENT_N",
                'VOLTAGE_ANGLE_L1', 'VOLTAGE_ANGLE_L2', 'VOLTAGE_ANGLE_L3', 
                "CURRENT_ANGLE_L1","CURRENT_ANGLE_L2","CURRENT_ANGLE_L3", 
                "POWER_FACTOR_L1","POWER_FACTOR_L2","POWER_FACTOR_L3","ACTIVE_POWER_L1","ACTIVE_POWER_L2","ACTIVE_POWER_L3","ACTIVE_POWER_TOTAL","KWH_ABS_TOTAL",
                'APPARENT_POWER_L1', 'APPARENT_POWER_L2', 'APPARENT_POWER_L3', 'BILL_REFF_KWH',
                'v_drop', 'v_lost', 'cos_phi_kecil', 'arus_hilang', 'In_more_Imax',
                'over_current', 'over_voltage', 'active_power_negative', 'active_power_negative_siang', 'active_power_negative_malam', 
                'unbalance_I', 'active_p_lost', 'arus_kecil_teg_kecil', 'current_loop', 'freeze', 'Jumlah Potensi TO', "SUM_WEIGHTED"]
    k1, k2, k3, k4 = st.columns(4)
    k1.download_button(
        "Download Data Siap TO",
        df_amr[SIAP_COL].to_csv(index=False, sep=';').encode('utf-8'),
        f"TO_AMR_{datetime_data_month}_{datetime_data_year}.csv",
        "text/csv",
        key='download-csv'
    )

    k2.download_button(
        "Download Data Full Features",
        df_amr[FULL_COL].to_csv(index=False, sep=';').encode('utf-8'),
        f"TO_full_AMR_{datetime_data_month}_{datetime_data_year}.csv",
        "text/csv",
        key='download-full-csv'
    )
# Load Profile Regular Operate
def operate_loadProfile():
    st.caption("Upload file operasional dulu, lalu pilih periode dan proses load profile.")

    uploaded_file = st.file_uploader(
        "Upload file Excel operating",
        type=["xls"],
        accept_multiple_files=False
    )

    if uploaded_file is None:
        st.info("Silakan upload file .xls terlebih dahulu.")
        return

    userInfo, err = errorHandling(lambda:read_excel(uploaded_file, 
                          nrows=6, 
                          usecols="A:B", 
                          header=None), "Berhasil Membaca Info Data Pelanggan")
    if err is not None:
        return
    
    labelInfo, valueInfo = userInfo.iloc[:, 0], userInfo.iloc[:, 1]

    INIT_COL = ["id", "Clock-time","L1 Last average voltage-value(V)","L2 Last  average voltage-value(V)","L3 Last  average voltage-value(V)","L1 Last average current-value(A)","L2 Last  average current-value(A)","L3 Last  average current-value(A)","last average import power factor-value","Last average ActiveImportPower-value(W)","Last average ActiveExportPower-value(W)","LP increment ActiveImportEnergyL1-value(Wh)","LP increment ActiveImportEnergyL2-value(Wh)","LP increment ActiveImportEnergyL3-value(Wh)","LP increment ActiveImportEnergy-value(Wh)","LP increment ActiveExportEnergy-value(Wh)"]
    DATA_TYPES={
        "id": "int64",
        "L1 Last average voltage-value(V)": "float64",
        "L2 Last  average voltage-value(V)": "float64",
        "L3 Last  average voltage-value(V)": "float64",
        "L1 Last average current-value(A)": "float64",
        "L2 Last  average current-value(A)": "float64",
        "L3 Last  average current-value(A)": "float64",
        "last average import power factor-value": "float64",
        "Last average ActiveImportPower-value(W)": "float64",
        "Last average ActiveExportPower-value(W)": "float64",
        "LP increment ActiveImportEnergyL1-value(Wh)": "float64",
        "LP increment ActiveImportEnergyL2-value(Wh)": "float64",
        "LP increment ActiveImportEnergyL3-value(Wh)": "float64",
        "LP increment ActiveImportEnergy-value(Wh)": "float64",
        "LP increment ActiveExportEnergy-value(Wh)": "float64",
    }
    parse_dates=["Clock-time"]
                   
    df, err = errorHandling(lambda:read_excel(uploaded_file, 
                                              skiprows=8, nrows=5077, usecols=INIT_COL, dtype=DATA_TYPES, parse_dates=parse_dates), "Berhasil Membaca Load Profile")
    if err is not None:
        return
    
    c1, c2 = st.columns([1, 2])
    with c1:
        st.metric("Nomor Meter", str(valueInfo[1]))
    with c2:
        st.metric("Jumlah Data", f"{len(df):,}")

    listOptionMonth, err = errorHandling(lambda:countPeriod(df["Clock-time"].dt.to_period("M")), "Berhasil Load Timestamp")
    if err is not None:
        return
    
    c1, c2 = st.columns([1, 1])
    with c1:
        selected = st.selectbox("Pilih Periode dari Data", listOptionMonth, index=0)
    with c2:
        e_m1_eis = st.number_input(
            f"Masukan Total Energi M1 (Dari EIS) - Pelanggan {valueInfo[1]} - Periode {selected}",
            min_value=0.0
        )

    c1, c2 = st.columns([1, 1])
    with c1:
        daya = st.number_input(f"Input Daya Pelanggan {valueInfo[1]}", min_value=0.0, value=900.0)
    with c2:
        fkm = st.number_input(
            "Input FKM",
            min_value=0.0, 
            value=60.0
        )

    process = st.button("Process")
    if process:
        dashboarding_profile(df[df["Clock-time"].dt.to_period("M") == selected],e_m1_eis, daya, fkm, selected)

def dashboarding_profile(df, e_m1_eis, daya, fkm, selected):
    df = df.copy()

    total_energi_m1 = df["LP increment ActiveImportEnergy-value(Wh)"].sum() * fkm / 1000
    total_m_energi_aktif = df["LP increment ActiveImportEnergy-value(Wh)"].sum() / 1000
    total_m_power_aktif = df["Last average ActiveImportPower-value(W)"].sum() / 1000
    total_energi_aktif_hitung = total_m_power_aktif * 15 / 60

    selisih_energi_m1 = e_m1_eis - total_energi_m1
    selisih_hitung = total_m_energi_aktif - total_energi_aktif_hitung
    persen_selisih_hitung = (selisih_hitung / total_m_energi_aktif * 100) if total_m_energi_aktif else None

    total_e_energi_aktif =  df["LP increment ActiveExportEnergy-value(Wh)"].sum() / 1000
    tegangan_maks = df[['L1 Last average voltage-value(V)', 'L2 Last  average voltage-value(V)', 'L3 Last  average voltage-value(V)']].max().max()
    peng = "TR" if tegangan_maks > 70 else "TM"

    # WBP / LWBP
    df["WBPOrLWBP"], err = errorHandling(
        lambda: WBPOrLWBP(df["Clock-time"]),
        "Berhasil Konversi WBP / LWBP"
    )
    if err is not None:
        st.error("Gagal konversi WBP / LWBP")
        return

    st.markdown("## Dashboard Load Profile")
    st.caption("Ringkasan energi, selisih, serta rata-rata tegangan dan arus per periode WBP/LWBP.")

    # Summary cards
    c1, c2, c3, c4 = st.columns(4)
    c1.metric("Total Energi M-1", f"{total_energi_m1:,.2f} kWh")
    c2.metric("EIS M-1", f"{e_m1_eis:,.2f} kWh")
    c3.metric("Selisih M-1", f"{selisih_energi_m1:,.2f} kWh")
    c4.metric("Import Power Aktif", f"{total_m_power_aktif:,.2f} kW")

    c5, c6, c7, c8 = st.columns(4)
    c5.metric("Import Energi Aktif", f"{total_m_energi_aktif:,.2f} kWh")
    c6.metric("Import Energi Aktif Hitung", f"{total_energi_aktif_hitung:,.2f} kWh")
    c7.metric(
        "Selisih Hitung",
        f"{selisih_hitung:,.2f} kWh",
        delta=f"{persen_selisih_hitung:.2f}%" if persen_selisih_hitung is not None else None
    )
    c8.metric("Jenis Pengukuran",  peng)

    st.divider()
    st.markdown("### Analisis Suspect")
    tab1, tab2, tab3 = st.tabs(["Energi", "Tegangan", "Arus"])

    wbp_mask = df["WBPOrLWBP"] == "WBP"
    lwbp_mask = df["WBPOrLWBP"] == "LWBP"

    with tab1:
        c1, c2, c3 = st.columns((3))
        c1.metric("Total Energi M-1", f"{total_energi_m1:,.2f} kWh")
        c2.metric("Total Energi EIS M-1", f"{e_m1_eis:,.2f} kWh")
        c3.metric("Selisih dengan EIS", f"{selisih_energi_m1:,.2f} kWh")
        
        c1, c2, c3 = st.columns((3))
        total_e_energi_aktif_wbp =  df.loc[wbp_mask, "LP increment ActiveExportEnergy-value(Wh)"].sum() / 1000
        total_e_energi_aktif_lwbp =  df.loc[lwbp_mask, "LP increment ActiveExportEnergy-value(Wh)"].sum() / 1000
        c1.metric("Total Energi Ekspor WBP", f"{total_e_energi_aktif_wbp:,.2f} kWh")
        c2.metric("Total Energi Ekspor LWBP", f"{total_e_energi_aktif_lwbp:,.2f} kWh")
        c3.metric("Total Energi Ekspor", f"{total_e_energi_aktif:,.2f} kWh")

        st.info(
            f"Perbandingan energi aktif hasil hitung terhadap total energi aktif: "
            f"{persen_selisih_hitung:.2f}%"
            if persen_selisih_hitung is not None else
            "Perbandingan energi aktif hasil hitung tidak tersedia."
        )

    with tab2:
        st.subheader("Rata-rata Tegangan")
        c1, c2 = st.columns(2)

        with c1:
            st.markdown("**WBP**")
            st.metric("R", fmt_num(safe_mean(df, wbp_mask, "L1 Last average voltage-value(V)")))
            st.metric("S", fmt_num(safe_mean(df, wbp_mask, "L2 Last  average voltage-value(V)")))
            st.metric("T", fmt_num(safe_mean(df, wbp_mask, "L3 Last  average voltage-value(V)")))

        with c2:
            st.markdown("**LWBP**")
            st.metric("R", fmt_num(safe_mean(df, lwbp_mask, "L1 Last average voltage-value(V)")))
            st.metric("S", fmt_num(safe_mean(df, lwbp_mask, "L2 Last  average voltage-value(V)")))
            st.metric("T", fmt_num(safe_mean(df, lwbp_mask, "L3 Last  average voltage-value(V)")))


    with tab3:
        wbp_mask = df["WBPOrLWBP"] == "WBP"
        lwbp_mask = df["WBPOrLWBP"] == "LWBP"

        st.subheader("Rata-rata Arus")
        c1, c2 = st.columns(2)

        with c1:
            st.markdown("**WBP**")
            st.metric("R", fmt_num(safe_mean(df, wbp_mask, "L1 Last average current-value(A)")))
            st.metric("S", fmt_num(safe_mean(df, wbp_mask, "L2 Last  average current-value(A)")))
            st.metric("T", fmt_num(safe_mean(df, wbp_mask, "L3 Last  average current-value(A)")))

        with c2:
            st.markdown("**LWBP**")
            st.metric("R", fmt_num(safe_mean(df, lwbp_mask, "L1 Last average current-value(A)")))
            st.metric("S", fmt_num(safe_mean(df, lwbp_mask, "L2 Last  average current-value(A)")))
            st.metric("T", fmt_num(safe_mean(df, lwbp_mask, "L3 Last  average current-value(A)")))

    # Optional: tampilkan ringkasan tabel kecil
    with st.expander("Lihat ringkasan angka mentah"):
        summary_df = pd.DataFrame([
            ["Total Energi M-1", total_energi_m1, "kWh"],
            ["EIS M-1", e_m1_eis, "kWh"],
            ["Selisih M-1", selisih_energi_m1, "kWh"],
            ["Import Energi Aktif", total_m_energi_aktif, "kWh"],
            ["Import Power Aktif", total_m_power_aktif, "kW"],
            ["Energi Aktif Hitung", total_energi_aktif_hitung, "kWh"],
        ], columns=["Metric", "Value", "Unit"])

        st.dataframe(summary_df, use_container_width=True, hide_index=True)
    
    # SUSPECT 
    ## V Drop 
    df["v_drop"], err = errorHandling(lambda:v_drop_lp(df), "Berhasil Klasifikasi V Drop")
    if err is not None:
        return
    
    ## I Loss
    df["i_loss"], err = errorHandling(lambda:i_loss_lp(df), "Berhasil Klasifikasi Arus Hilang")
    if err is not None:
        return
    
    # Freeze 
    df["freeze"], err = errorHandling(lambda:freeze_lp(df), "Berhasil Klasifikasi Freeze")
    if err is not None:
        return
    
    suspect_cols = ["v_drop", "i_loss", "freeze"]
    df["jumlah"] = df[suspect_cols].sum(axis=1)

    summary = df[suspect_cols].sum().astype(int)
    total_rows = len(df)
    total_flags = int(summary.sum())

    c1, c2, c3, c4 = st.columns(4)
    c1.metric("Total Data", f"{total_rows:,}")
    c2.metric("V Drop", f"{summary['v_drop']:,}")
    c3.metric("I Loss", f"{summary['i_loss']:,}")
    c4.metric("Freeze", f"{summary['freeze']:,}")

    st.divider()
    top_issue = summary.idxmax()
    top_count = int(summary.max())

    col_a, col_b = st.columns([2, 1])
    with col_a:
        st.success(
            f"Terdapat total **{total_flags:,}** indikasi suspect dari **{total_rows:,}** baris data. "
            f"Indikasi paling dominan adalah **{top_issue.upper()}** sebanyak **{top_count:,}** kejadian."
        )

    with col_b:
        st.info(
            f"Jumlah baris terindikasi minimal 1 suspect: "
            f"**{int((df['jumlah'] > 0).sum()):,}**"
        )

    tab1, tab2 = st.tabs(["Detail", "Ringkasan Suspect"])

    with tab1:
        df_table = df[["Clock-time", "v_drop", "i_loss", "freeze", "jumlah"]].copy()

        def highlighter(val):
            if val == 1:
                return "background-color: #ffe5e5; color: #b30000; font-weight: bold;"
            return ""

        styled_df = df_table.style.applymap(highlighter, subset=["v_drop", "i_loss", "freeze"]).format({
                "Clock-time": lambda x: x.strftime("%Y-%m-%d %H:%M:%S") if pd.notna(x) else "-",
                "jumlah": "{:.0f}"
            })

        st.dataframe(
            styled_df,
            use_container_width=True,
            height=500,
            hide_index=True
        )

    with tab2:
        summary_df = pd.DataFrame({
            "Suspect": ["V Drop", "I Loss", "Freeze"],
            "Jumlah": [int(summary["v_drop"]), int(summary["i_loss"]), int(summary["freeze"])],
            "Persentase": [
                f"{(summary['v_drop'] / total_rows * 100):.2f}%" if total_rows else "0.00%",
                f"{(summary['i_loss'] / total_rows * 100):.2f}%" if total_rows else "0.00%",
                f"{(summary['freeze'] / total_rows * 100):.2f}%" if total_rows else "0.00%",
            ]
        })

        st.dataframe(summary_df, use_container_width=True, hide_index=True)

        st.markdown("### Interpretasi")
        if summary["i_loss"] >= summary["v_drop"] and summary["i_loss"] >= summary["freeze"]:
            st.write("I Loss adalah suspect yang paling sering muncul pada periode ini.")
        elif summary["v_drop"] >= summary["freeze"]:
            st.write("V Drop adalah suspect yang paling sering muncul pada periode ini.")
        else:
            st.write("Freeze adalah suspect yang paling sering muncul pada periode ini.")

        if (df["jumlah"] > 0).any():
            st.write(
                f"Ada **{int((df['jumlah'] > 0).sum()):,}** baris yang terindikasi minimal satu gangguan suspect."
            )
        else:
            st.write("Tidak ada indikasi suspect pada periode ini.")
    
    ts = int(time.time())
    # Create Excel in memory
    buffer = io.BytesIO()

    with pd.ExcelWriter(buffer, engine="openpyxl") as writer:
        df.to_excel(writer, index=False)

    # IMPORTANT
    excel_data = buffer.getvalue()

    st.download_button(
        label="Download Data Full Features",
        data=excel_data,
        file_name=f"TO_full_LP_{selected}_{ts}.xlsx",
        mime="application/vnd.openxmlformats-officedocument.spreadsheetml.sheet"
    )

# Shared utility (Pasca & Prabayar) - konversi kode baca ke deskripsi
def convert_kode_baca(kode_baca):
    switcher = {"1":"KWH METER TINGGI","2":"GANTI KWH METER","3":"ANJING GALAK","7":"STAN DARI PELANGGAN","D":"PELANGGAN TIDAK SESUAI RBM","E":"RUMAH TUTUP","F":"RUMAH KOSONG","I":"KWH METER DIDALAM BANGUNAN","L":"TARIF TIDAK SESUAI","M":"KWH KURANG TAGIH","N":"KWH LEBIH TAGIH","U":"PEMADAMAN","W":"BENCANA","A":"KWH METER MACET","B":"KWH METER MUNDUR","C":"KWH METER TIDAK ADA","H":"KWH METER BURAM/PECAH","J":"SEGEL TIDAK ADA","K":"MCB PERLU DIPERIKSA","Z":"NORMAL",}
    return switcher.get(kode_baca, None)
# Pasca Properties 
def jenis_pelanggan(HB):
    if pd.isna(HB):
        return "AMR"
    elif( HB == "I"):
        return "AMI"
    else:
        return "Non-AMR"

# Pasca Properties 
def further_steps(month, year, c2_pasca, prev_month):
    further_cond = False
    df_pakai_tahun_lalu = None
    this_month_year = None
    read_year = None
    # File Loader Step 2
    uploaded_input_step_add  = c2_pasca.file_uploader(
        f"Additional Step. Inputkan Data Pembacaan Meter {month}-{year} (x-12+1)", type=["txt"]
    )
    placeholder3 = c2_pasca.empty()
    placeholder3 = c2_pasca.warning('⚠️ Data Pembacaan Meter Belum di Upluod')
    dict_input = {
        "IDPEL": [], 
        "PEM KWH TAHUN LALU": []
    }
    if uploaded_input_step_add:
        lines = uploaded_input_step_add.readlines()
        if lines:
            try:
                line_temp = lines[0].decode("utf-8")
            except UnicodeDecodeError:
                try:
                    line_temp = lines[0].decode("utf-8-sig")
                except UnicodeDecodeError:
                    line_temp = lines[0].decode("latin-1")
            tahun_bulan_add = line_temp[338:344]
            
            for line in lines:
                try:
                    line = line.decode("utf-8")
                except UnicodeDecodeError:
                    try:
                        line = line.decode("utf-8-sig")
                    except UnicodeDecodeError:
                        line = line.decode("latin-1")
                if len(line) >= 362:
                    dict_input["IDPEL"].append(line[0:12].strip())
                    dict_input["PEM KWH TAHUN LALU"].append(int(line[298:310].strip()))

        # Ambil Satu Titik Bulan 
        df_pakai_tahun_lalu = pd.DataFrame.from_dict(dict_input)
        this_month_year, _, _ = decode_month(tahun_bulan_add[-2:])
        read_year = tahun_bulan_add[:-2]
        if (this_month_year != prev_month) | (read_year != year):
            placeholder3.warning(f"⚠️ Sistem kami mendeteksi anda meng-upload Data Pembacaan Meter {this_month_year}-{read_year}, seharusnya anda meng-upload {month}-{year} !")
            further_cond = True
        else:
            placeholder3.success(f"✅ Data Tambahan telah Selesai di Proses.")
            further_cond = True
    return further_cond, df_pakai_tahun_lalu, this_month_year, read_year
# Pasca Dashboard
def dashboarding_pasca(df_pasca, prev_month, tahun, forth_prev_month, third_prev_month, two_prev_month, opt_cond, add_month, add_year):
    st.markdown(f"""<h2>Dashboard Potensi Target Operasi P2TL Pascabayar Non AMR Bulan {prev_month}</h2>""", unsafe_allow_html=True)
    df_pasca['IDPEL'] = df_pasca['IDPEL'].astype(str)
    df_pasca.loc[:, ['ST AWAL', 'ST AKHIR', 'PEM KWH BULAN KETIGA', 'PEM KWH BULAN KEDUA', 'PEM KWH BULAN PERTAMA',  'PEM KWH BULAN 0']] = df_pasca[['ST AWAL', 'ST AKHIR', 'PEM KWH BULAN KETIGA', 'PEM KWH BULAN KEDUA', 'PEM KWH BULAN PERTAMA',  'PEM KWH BULAN 0']].astype(float)
    df_pasca['RERATA'] = df_pasca[["PEM KWH BULAN 0", 'PEM KWH BULAN PERTAMA', 'PEM KWH BULAN KEDUA']].mean(axis=1)
    df_pasca['Potensi Temuan (kWh)'] = (df_pasca['PEM KWH BULAN KETIGA']- df_pasca['RERATA'])
    ## metrics
    # Jumlah Customer 
    n_cust = len(df_pasca)
    n_amr = len(df_pasca[df_pasca['KDBACA'].isnull()])
    n_namr = n_cust - n_amr
    # Proyeksi Customer Baru
    prev_months = ["PEM KWH BULAN 0", 'PEM KWH BULAN PERTAMA','PEM KWH BULAN KEDUA']
    n_new_cust = len(df_pasca[(((df_pasca[prev_months] == 0) | df_pasca[prev_months].isna()).all(axis=1)) & (df_pasca['PEM KWH BULAN KETIGA'] > 0)])
    # Pelanggan Aktif tiap bulan
    n_0 = len(df_pasca["PEM KWH BULAN 0"]) - len(df_pasca[df_pasca["PEM KWH BULAN 0"].isna() | df_pasca["PEM KWH BULAN 0"] == 0])
    n_1 = len(df_pasca['PEM KWH BULAN PERTAMA']) - len(df_pasca[df_pasca['PEM KWH BULAN PERTAMA'].isna() | df_pasca['PEM KWH BULAN PERTAMA'] ==0])
    n_2 = len(df_pasca['PEM KWH BULAN KEDUA']) - len(df_pasca[df_pasca['PEM KWH BULAN KEDUA'].isna() | df_pasca['PEM KWH BULAN KEDUA'] ==0])
    n_3 = len(df_pasca['PEM KWH BULAN KETIGA']) - len(df_pasca[df_pasca['PEM KWH BULAN KETIGA'].isna() | df_pasca['PEM KWH BULAN KETIGA'] ==0])

    # Kwh Terpakai
    data_m = [df_pasca["PEM KWH BULAN 0"].sum(), df_pasca['PEM KWH BULAN PERTAMA'].sum(), df_pasca['PEM KWH BULAN KEDUA'].sum(), df_pasca['PEM KWH BULAN KETIGA'].sum()]
    c1_pasca, c2_pasca, c3_pasca, c4_pasca = st.columns(4)
    c1_pasca.metric(label="Jumlah Pelanggan Pascabayar", value=n_cust, delta=f"{n_3} (Active)\nAMR: {n_amr}\tNon-AMR: {n_namr}", delta_color='off')
    c2_pasca.metric(label="Estimasi Besar Pelanggan Baru", value=n_new_cust, delta=f"{n_new_cust/(n_cust-n_new_cust)*100:.2f}%")
    c3_pasca.metric(label="KwH terpakai bulan terbaru", value = f"{data_m[-1]:,.0f}", delta=f"Penurunan dari Rerata (kWh): {df_pasca['Potensi Temuan (kWh)'].sum():,.1f}", delta_color='off')

    m = [n_0, n_1, n_2, n_3]
    bulan = ["PEM KWH BULAN 0", "PEM KWH BULAN PERTAMA", "PEM KWH BULAN KEDUA", "PEM KWH BULAN KETIGA"]
    rename_bulan = [forth_prev_month, third_prev_month, two_prev_month, prev_month]
    rerata_pemakaian = [data_m[i] / m[i] for i in range(4)]

    temp = pd.DataFrame({
        'Bulan': rename_bulan,
        'Total_kWh': data_m,
        'Rerata_kWh': rerata_pemakaian
    })

    # Chart batang (Rerata)
    bar = alt.Chart(temp).mark_bar(size=30, color='red', opacity=0.6).encode(
        x=alt.X('Bulan:N', title='Bulan', axis=alt.Axis(labelAngle=0, labelFontSize=12)),
        y=alt.Y('Rerata_kWh:Q', title='Rerata kWh', axis=alt.Axis(titleColor='red')),
        tooltip=['Bulan', 'Rerata_kWh']
    )

    # Chart garis (Total)
    line = alt.Chart(temp).mark_line(point=True, color='blue').encode(
        x=alt.X('Bulan:N'),
        y=alt.Y('Total_kWh:Q', title='Total kWh', axis=alt.Axis(titleColor='blue')),
        tooltip=['Bulan', 'Total_kWh']
    )

    # Gabungkan chart dan pisahkan sumbu Y
    chart = alt.layer(bar, line).resolve_scale(
        y='independent'  # Biar punya dua sumbu Y (kiri dan kanan)
    ).properties(
        title='Total Pemakaian kWh dan Rerata per Bulan',
        width='container'
    ).configure_title(
        anchor='middle',
        fontSize=16
    )

    st.altair_chart(chart, use_container_width=True)

    # SETTINGAN PARAMETER
    with st.expander("Setting Parameter 📐"):
        # FILTER
        co1, co2, co3, co4, co5 = st.columns((1, 1, 2, 1, 1))
        with co1: 
            batas_wajar = st.number_input(label="Set Batas Penurunan Kewajaran (%)*", min_value=10, max_value=200, value=70)
            df_pasca['decrease'] = df_pasca['Potensi Temuan (kWh)'] < (batas_wajar*-1/100)*df_pasca['RERATA']
        with co2:
            def_kdbc = ['NORMAL']
            try:
                list_kdbc = st.multiselect(label="Keterangan Kode Baca*", options=df_pasca['KDBACA'].unique(), default=def_kdbc)
            except Exception as e:
                list_kdbc = st.multiselect(label="Keterangan Kode Baca*", options=df_pasca['KDBACA'].unique())
        with co3: 
            # list_daya = st.multiselect(label="Pilihan Daya", options=list(df.DAYA.unique()), default=list(df.DAYA.unique()))
            min_val, max_val = min(df_pasca.DAYA.unique()), max(df_pasca.DAYA.unique())
            min_daya, max_daya = st.slider(label="Range Daya*", min_value=min_val, max_value=max_val, value=(min_val, max_val))
        with co4: 
            number_show = st.number_input(label="Banyak Data yang Ingin Ditampilkan*", min_value=0, value=25)
        with co5: 
            jenis_pel = st.multiselect(label="Jenis Pelanggan", options=df_pasca['KdPM'].unique())
            if jenis_pel:
                df_pasca=df_pasca[df_pasca['KdPM'].isin(jenis_pel)]
        st.caption("*Parameter harus diisi")

        # applied filtering
        # filtering about daya
        df_pasca = df_pasca[(df_pasca['DAYA']>=min_daya)&(df_pasca['DAYA']<=max_daya)]
        # filtering about kode baca
        df_pasca = df_pasca[df_pasca['KDBACA'].isin(list_kdbc)]
        df_pasca = df_pasca[df_pasca['decrease']==True]

    # Metric Temuan
    potensi_kwh_hilang = round(df_pasca['Potensi Temuan (kWh)'].sum())
    c4_pasca.metric(label="Potensi Kasus TO", value=len(df_pasca), delta=f"Potensi kWh Hilang: {potensi_kwh_hilang}", delta_color="off")

    st.subheader(f"Top {number_show} Rekomendasi Target Operasi")
    IMPORTANT_COL = ['IDPEL', 'KdPM', 'NAMA PELANGGAN', 'ALAMAT', 'TARIF', 'DAYA', 'PEM KWH BULAN KETIGA', 'RERATA', 'Potensi Temuan (kWh)', 'KDBACA']
    if opt_cond:
        df_pasca['DeviasiTahunLalu'] = df_pasca['PEM KWH BULAN KETIGA'] - df_pasca['PEM KWH TAHUN LALU']
        IMPORTANT_COL = ['IDPEL', 'KdPM', 'NAMA PELANGGAN', 'ALAMAT', 'TARIF', 'DAYA', 'PEM KWH BULAN KETIGA', 'RERATA', 'Potensi Temuan (kWh)', 'PEM KWH TAHUN LALU', 'DeviasiTahunLalu', 'KDBACA']
        renames_ = {'PEM KWH BULAN KETIGA': f"{prev_month[:3]}-{tahun[2:]}", 'PEM KWH TAHUN LALU': f"{add_month[:4]}-{add_year[2:]}"}
    else:
        renames_ = {'PEM KWH BULAN KETIGA': f"{prev_month[:3]}-{tahun[2:]}"}
    st.table(
        df_pasca.sort_values(by='Potensi Temuan (kWh)', ascending=True)[IMPORTANT_COL].head(number_show).rename(renames_, axis=1).reset_index(drop=True).style.format({
            'RERATA': '{:.1f}',
            'Potensi Temuan (kWh)': '{:.1f}'
        })
    )
    # Buat Download CSV
    st.download_button(
        "Tekan untuk Download Data",
        df_pasca[IMPORTANT_COL].to_csv(index=False, sep=';').encode('utf-8'),
        "hasil_analisis.csv",
        "text/csv",
        key='download-csv'
    )

# Pasca Operation 
def operate_pasca():
    st.markdown("""
    <h3>Otomatisasi Target Operasi <strong>Pelanggan Pascabayar</strong></h4>
    <p>
    Untuk Menghasilkan Target Operasi Bulan (x) pada Pascabayar dibutuhkan 3 data, diantaranya: 
    <li>Data Pembacaan Meter Bulan Depan (x+1), <a href="https://drive.google.com/file/d/1EydEwklnhxcPlwzyKv4VpWxd4Lp5gobh/view?usp=drive_link">Contoh Data Pembacaan Meter Bulan Depan (x+1)</a></li>
    <li>Data Pembacaan Meter Bulan Ini (x) , <a href="https://drive.google.com/file/d/1935WDaxFIwZy363ccURuYb78NgfjGNQF/view?usp=drive_link">Contoh Data Pembacaan Meter Bulan Ini (x)</a></li></li>
    <li>Data Customer Satu Bulan sebelumnya (x-1) , <a href="https://docs.google.com/spreadsheets/d/1pFSm2kOpfYkKaOi8RkfLzY6AWqfVqwSo/edit?usp=drive_link&ouid=113680444123721717153&rtpof=true&sd=true">Contoh Data Customer (x-1)</a></li></li>
    Berdasarkan data tersebut, akan dihasilkan TO Bulan (x) berdasar trend konsumsi listrik pelanggan bulan (x) dibandingkan tiga bulan sebelum itu (x-1, x-2, x-3). Kami akan mengambil Keterangan Kode Baca dari Data Customer berdasar hasil baca saat bulan sebelum TO (x-1). 
    </p>
    """, unsafe_allow_html=True)
    with st.expander(label="Input Data"):
        c1_pasca, c2_pasca, c4_pasca = st.columns(3)
        # File Loader Step 1
        uploaded_input_step1  = c1_pasca.file_uploader(
            "Step 1. Inputkan Data Pembacaan Meter Bulan Depan (x+1)", type=["txt"]
        )
        placeholder1 = c1_pasca.empty()
        placeholder1 = c1_pasca.warning('⚠️ Data Pembacaan Meter Belum di Upluod')
        # File Loader Step 2
        uploaded_input_step2  = c2_pasca.file_uploader(
            "Step 2. Inputkan Data Pembacaan Meter Bulan Ini (x)", type=["txt"]
        )
        placeholder2 = c2_pasca.empty()
        placeholder2 = c2_pasca.warning('⚠️ Data Pembacaan Meter Belum di Upluod')
        # File Loader Step 3
        uploaded_input_step4  = c4_pasca.file_uploader(
            "Step 3. Inputkan Data Customer Bulan (x-1)", type=["csv", "xlsx", "xls"]
        )
        placeholder4 = c4_pasca.empty()
        placeholder4 = c4_pasca.markdown(
            """💡 Tips: <a href="https://ludyhasby.github.io/inovasi_menteng/" target='_blank'>Konversi ke CSV</a> supaya efisien""",
            unsafe_allow_html=True
        )

    if uploaded_input_step1:
        dict_input = {
            "IDPEL": [], "NAMA PELANGGAN": [], "ALAMAT": [], "TARIF": [], "DAYA": [],
            "KDDK": [], "RBM": [], "T/G": [], "ST AWAL": [], "ST AKHIR": [],
            "PEM KWH BULAN KETIGA": [], "PEM KWH BULAN KEDUA": [], "PEM KWH BULAN PERTAMA": [],
            "TAHUN_BULAN": [], "KODE KELOMPOK": [], "TGL BACA BULAN KEDUA": [], "TGL BACA BULAN KETIGA": []
        }

        for line in uploaded_input_step1.readlines():
            try:
                line = line.decode("utf-8")
            except UnicodeDecodeError:
                try:
                    line = line.decode("utf-8-sig")
                except UnicodeDecodeError:
                    line = line.decode("latin-1")

            if (len(line) >= 362) & (len(line) < 390):
                dict_input["IDPEL"].append(str(line[0:12].strip()))
                dict_input["NAMA PELANGGAN"].append(line[12:37].strip())
                dict_input["ALAMAT"].append(line[37:67].strip())
                dict_input["TARIF"].append(line[67:70].strip())
                dict_input["DAYA"].append(int(line[71:80].strip()))
                dict_input["KDDK"].append(line[80:88].strip())
                dict_input["RBM"].append(line[88:91].strip())
                dict_input["T/G"].append(line[93:94].strip())
                dict_input["ST AWAL"].append(int(line[157:175].strip()))
                dict_input["ST AKHIR"].append(int(line[206:212].strip()))
                dict_input["PEM KWH BULAN KETIGA"].append(int(line[298:310].strip()))
                dict_input["PEM KWH BULAN KEDUA"].append(int(line[310:322].strip()))
                dict_input["PEM KWH BULAN PERTAMA"].append(int(line[322:334].strip()))
                dict_input["TAHUN_BULAN"].append(line[338:344].strip())
                dict_input["KODE KELOMPOK"].append(line[344:346].strip())
                dict_input["TGL BACA BULAN KEDUA"].append(line[346:354].strip())
                dict_input["TGL BACA BULAN KETIGA"].append(line[354:362].strip())
            elif (len(line) >= 390):
                dict_input["IDPEL"].append(str(line[0:12].strip()))
                dict_input["NAMA PELANGGAN"].append(line[12:37].strip())
                dict_input["ALAMAT"].append(line[37:67].strip())
                dict_input["TARIF"].append(line[67:70].strip())
                dict_input["DAYA"].append(int(line[71:80].strip()))
                dict_input["KDDK"].append(line[80:88].strip())
                dict_input["RBM"].append(line[88:91].strip())
                dict_input["T/G"].append(line[93:94].strip())
                dict_input["ST AWAL"].append(int(line[157:175].strip()))
                dict_input["ST AKHIR"].append(int(line[206:212].strip()))
                dict_input["PEM KWH BULAN KETIGA"].append(int(line[334:346].strip()))
                dict_input["PEM KWH BULAN KEDUA"].append(int(line[346:358].strip()))
                dict_input["PEM KWH BULAN PERTAMA"].append(int(line[358:370].strip()))
                dict_input["TAHUN_BULAN"].append(line[374:380].strip())
                dict_input["KODE KELOMPOK"].append(line[380:382].strip())
                dict_input["TGL BACA BULAN KEDUA"].append(line[382:390].strip())
                dict_input["TGL BACA BULAN KETIGA"].append(line[390:398].strip())

        # Ambil Satu Titik Bulan 
        tahun_bulan = dict_input["TAHUN_BULAN"][0][-2:]
        tahun = dict_input["TAHUN_BULAN"][0][:-2]
        df_pasca = pd.DataFrame.from_dict(dict_input)
        this_month, prev_month, two_prev_month = decode_month(tahun_bulan[:2])
        third_prev_month, forth_prev_month = prev_three_four(tahun_bulan[:2])

        placeholder1.success(f"✅ Data 1 telah Selesai di Proses. Anda menginputkan Data {this_month}. Dengan demikian, TO {prev_month} akan di-generate 😊.")

    if uploaded_input_step2:
        if uploaded_input_step1 is None:
            placeholder2.error("Harap Upload Step 1 terlebih dahulu sebelum Step 2!")
        else:
            dict_input = {
                "IDPEL": [], 
                "PEM KWH BULAN 0": []
            }
            lines = uploaded_input_step2.readlines()
            if lines:
                try:
                    line_temp = lines[0].decode("utf-8")
                except UnicodeDecodeError:
                    try:
                        line_temp = lines[0].decode("utf-8-sig")
                    except UnicodeDecodeError:
                        line_temp = lines[0].decode("latin-1")
                tahun_bulan1 = line_temp[338:344]             
                for line in lines:
                    try:
                        line = line.decode("utf-8")
                    except UnicodeDecodeError:
                        try:
                            line = line.decode("utf-8-sig")
                        except UnicodeDecodeError:
                            line = line.decode("latin-1")
                    if (len(line) >= 362) & (len(line) < 390):
                        dict_input["IDPEL"].append(str(line[0:12].strip()))
                        dict_input["PEM KWH BULAN 0"].append(int(line[322:334].strip()))
                    elif len(line) > 390: 
                        # jika karakter > 390, benerin tahun bulannya
                        tahun_bulan1 = line_temp[374:380]  
                        dict_input["IDPEL"].append(str(line[0:12].strip()))
                        dict_input["PEM KWH BULAN 0"].append(int(line[358:370].strip()))
            # Ambil Satu Titik Bulan 
            df_pakai_lalu = pd.DataFrame.from_dict(dict_input)
            this_month1, _, _ = decode_month(tahun_bulan1[-2:])
            if this_month1 != prev_month:
                placeholder2.warning(f"⚠️ Pastikan anda meng-upload {prev_month} ya 😊")
            else:
                placeholder2.success(f"✅ Data 2 telah Selesai di Proses.")
            opt_step = c2_pasca.toggle(label="Fitur Tambahan: Track dengan Pemakaian Tahun Lalu (x-12)")
            opt_cond = False
            if opt_step: 
                opt_cond, df_pakai_tahun_lalu, add_month, add_year = further_steps(this_month, str(int(tahun)-1), c2_pasca, prev_month)
            else:
                add_month, add_year = None, None

    if uploaded_input_step4:
        filename = uploaded_input_step4.name.lower()
        if filename.endswith(".csv"):
            try: 
                df_cust = pd.read_csv(uploaded_input_step4, sep=';')
            except Exception as e:
                try:
                    encodings = ["utf-8", "utf-8-sig", "latin-1", "cp1252"]
                    df_cust = read_csv_try(
                        uploaded_input_step4,
                        enc_list=encodings,
                        sep=";",
                        low_memory=False,
                    )
                except Exception as e:
                    c4_pasca.warning("Gagal Membaca Data")
        else:
            df_cust = read_excel(uploaded_input_step4)
        df_cust= df_cust[df_cust['IDPEL'].notna()]
        df_cust["IDPEL"] = df_cust["IDPEL"].apply(lambda x: str(x).split('.')[0])
        df_cust['BLTH'] = df_cust['BLTH'].astype(int, errors='ignore')

        tahun_bulan = str(df_cust.loc[0, "BLTH"])[-2:]
        this_month_cust, prev_month_cust, two_prev_month_cust = decode_month(tahun_bulan)
        df_cust['IDPEL'] = df_cust['IDPEL'].astype(str)
        try: 
            df_cust = df_cust[['IDPEL', 'KDBACA', 'HB']]
        except Exception as e:
            try: 
                df_cust = df_cust[['IDPEL', 'KDBACA', 'HARI BACA']]
                df_cust.rename(columns={'HARI BACA': 'HB'}, inplace=True)
            except Exception as e:
                df_cust = df_cust[['IDPEL', 'KDBACA']]
                df_cust['HB'] = None

        if this_month_cust == prev_month:
            placeholder4.success(f"✅ Data 3 telah Selesai di Proses.")
        else:
            placeholder4.warning(f"⚠️ Sistem kami mendeteksi anda meng-upload Data Customer {prev_month_cust}, seharusnya anda meng-upload {two_prev_month} !")

    placeholder = st.empty()
    if uploaded_input_step1 and uploaded_input_step2 and uploaded_input_step4:
        placeholder.warning("Data telah diterima, selanjutnya akan kami proses dan Data Pembacaan Meterakukan analisis ⚒️⚒️")

        # Penggabungan
        df_pasca = df_pasca.merge(df_pakai_lalu, how='inner', on='IDPEL')
        df_pasca["PEM KWH BULAN 0"] = df_pasca["PEM KWH BULAN 0"].fillna(0).astype('int')
        # step tambahan 
        if opt_cond: 
            df_pasca = df_pasca.merge(df_pakai_tahun_lalu, how='inner', on='IDPEL')
            df_pasca["PEM KWH TAHUN LALU"] = df_pasca["PEM KWH TAHUN LALU"].fillna(0).astype('int')

        # pakai penggabungan
        df_pasca = pd.merge(df_pasca, df_cust, how='left', on='IDPEL')
        df_pasca["KDBACA"] = df_pasca['KDBACA'].apply(convert_kode_baca)
        df_pasca["KdPM"] = df_pasca['HB'].apply(jenis_pelanggan)
        dashboarding_pasca(df_pasca, prev_month, tahun, forth_prev_month, third_prev_month, two_prev_month, opt_cond, add_month, add_year)
        placeholder.empty()
        placeholder.success("🎉 Hasil Analisa sudah keluar !")
        st.divider()

# Prabayar Properties
def get_bulan_0(month_year_string):
    year = int(month_year_string[:4])
    month = int(month_year_string[4:])
    sum_m = year*12+month-4
    fin_year = sum_m//12
    fin_month = sum_m%12
    if fin_month < 10:
        return f"0{fin_month}_{fin_year}"
    else:
        return f"{fin_month}_{fin_year}"

def weight_ct(number_ct):
    if number_ct == 1:
        return 0.5
    elif number_ct == 2:
        return 1
    elif number_ct == 3:
        return 2
    elif number_ct > 3:
        return 3
    return 0
def weight_token(string_token):
    if string_token == 'dalam 1 bulan tidak beli token':
        return 1
    elif string_token == 'dalam 2 sd 3 bulan berturut-turut tidak beli token':
        return 2
    elif string_token == 'dalam 4 sd 6 bulan berturut-turut tidak beli token':
        return 2
    elif string_token == 'dalam > 6 bulan berturut-turut dan seterusnya tidak beli token':
        return 3
    return 0
    
# Prabayar Properties
def dashboarding_pra(df_pra):
    df_pra["TEGANGAN"] = df_pra["TEGANGAN"].astype(float)
    tahun_bulan = str(int(df_pra.iloc[0]["BLTH"]))[-2:]
    next_month, this_month, _ = decode_month(tahun_bulan)
    st.markdown(f"""<h2>Dashboard Target Operasi P2TL Prabayar Bulan {this_month}</h2>""", unsafe_allow_html=True)
    n_data = len(df_pra)
    df_pra['Jumlah Potensi TO'] = 0

    # SETTINGAN PARAMETER
    with st.expander("Setting Parameter 📐"):
        st.write("Operasi Logika yang digunakan disini a(dalah OR. Dengan demikian, jika anda ingin melihat dari satu indikator saja, matikan indikator lain dan rubah nilai indikator sesuai keinginan. Indikator yang sesuai dengan operasi akan di highlight berwarna hijau cerah dan berkontribusi pada perhitungan potensi TO.")
        # FILTER
        co1, co2, co3, co4 = st.columns((1, 1, 2, 1))
        with co1: 
            batas_wajar = st.number_input(label="Set Batas Atas Tegangan (V) | Tegangan < ", value=198)       
            conditions_tegangan = (df_pra["TEGANGAN"] < batas_wajar)
            df_pra['tegangan_kecil'] = conditions_tegangan.fillna(False)
            
            keypad_options = df_pra["KEYPAD"].unique()
            default_value = ['Rusak'] if 'Rusak' in keypad_options else []
            selection_keypad = st.multiselect(
                label="Kondisi Keypad", 
                options=keypad_options, 
                default=default_value
            )
            conditions_keypad = (df_pra["KEYPAD"].isin(selection_keypad))
            df_pra['keypad_bermasalah'] = conditions_keypad.fillna(False)
        with co2:
            batas_atas_cosphi = st.number_input(label="Set Batas Atas cos phi | cos phi < ", value=0.85)
            conditions_cosphi = (df_pra['COSPHI'] < batas_atas_cosphi)
            df_pra['cosphi_kecil'] = conditions_cosphi.fillna(False)

            indi_temper_list = df_pra["INDI_TEMPER"].unique()
            temper_default = ['Nyala'] if "Nyala" in indi_temper_list else []
            selection_ind_temper = st.multiselect(label="Indikator Temper", options=indi_temper_list, default=temper_default)
            conditions_ind_temper = (df_pra['INDI_TEMPER'].isin(selection_ind_temper))
            df_pra['indikator_temper'] = conditions_ind_temper.fillna(False)
        with co3: 
            segel_list = df_pra["SEGEL"].unique()
            segel_default = ['Tidak Ada'] if 'Tidak Ada' in segel_list else []
            selection_segel = st.multiselect(label="Kondisi Segel", options=segel_list, default=segel_default)
            conditions_segel = (df_pra['SEGEL'].isin(selection_segel))
            df_pra['segel_bermasalah'] = conditions_segel.fillna(False)

            relay_list = df_pra["RELAY"].unique()
            relay_default = ['Buka'] if 'Buka' in relay_list else []
            selection_relay = st.multiselect(label="Opsi Relay", options=relay_list, default=relay_default)
            conditions_relay = (df_pra['RELAY'].isin(selection_relay))
            df_pra['opsi_relay'] = conditions_relay.fillna(False)
        with co4: 
            lcd_list = list(df_pra["LCD"].unique())
            # FIX: list.remove() returns None, gunakan list comprehension
            lcd_list_default = [x for x in lcd_list if x != "Normal"]
            selection_lcd = st.multiselect(label="Kondisi LCD", options=lcd_list, default=lcd_list_default)
            conditions_lcd = (df_pra['LCD'].isin(selection_lcd))
            df_pra['LCD_bermasalah'] = conditions_lcd.fillna(False)
        
        st.subheader(" ", divider='green')
        k1, k2 = st.columns(2)
        k1.write("Untuk menentukan Target Operasi (TO), perlu di tentukan batas minimum kriteria yang dipenuhi, misalnya 1 dari 7 indikator terpenuhi ✅. ")
        n_indikator = k1.number_input(label="Jumlah Indikator >= ", min_value=1, value=1, max_value=7)    
        n_show = k2.number_input(label='Banyak data yang ingin ditampilkan', value=25)    
        options_params = ['tegangan_kecil', 'keypad_bermasalah', 'indikator_temper', 'segel_bermasalah', 'opsi_relay', 'LCD_bermasalah']
        # handling if Clear Temper
        if 'JML_CT' in df_pra.columns:
            options_params.append('clear_temper')
            df_pra['CT_weight'] = df_pra['JML_CT'].apply(weight_ct)
            df_pra['clear_temper'] = df_pra['CT_weight'] > 0
        # handling not buy a token
        if 'Keterangan_Token' in df_pra.columns:
            options_params.append('tidak_beli_token')
            df_pra['token_weight'] = df_pra['Keterangan_Token'].apply(weight_token)
            df_pra['tidak_beli_token'] = df_pra['token_weight'] > 0

        selected_options = k2.multiselect(label='Kriteria Wajib Terpenuhi (Opsional)', options=options_params)
        if selected_options:
            k2.caption("Jika Dipilih, Operasi nya kira-kira menjadi: 'AND([Kriteria_Wajib_i]) * OR([Kriteria_Tambahan_i])'")
            df_pra = df_pra[df_pra[selected_options].eq(True).all(axis=1)]

    # applied filtering
    def tegangan_rendah(val):
        color = '#D3F6EC' if val < batas_wajar else 'transparent'
        return f'background-color: {color}'
    def cosphi_rendah(val):
        color = '#D3F6EC' if val < batas_atas_cosphi else 'transparent'
        return f'background-color: {color}'
    def segel_highlighter(val):
        color = '#D3F6EC' if val in selection_segel else 'transparent'
        return f'background-color: {color}'
    def lcd_highlighter(val):
        color = '#D3F6EC' if val in selection_lcd else 'transparent'
        return f'background-color: {color}'
    def keypad_highlighter(val):
        color = '#D3F6EC' if val in selection_keypad else 'transparent'
        return f'background-color: {color}'
    def indi_temper_highlighter(val):
        color = '#D3F6EC' if val in selection_ind_temper else 'transparent'
        return f'background-color: {color}'
    def relay_highlighter(val):
        color = '#D3F6EC' if val in selection_relay else 'transparent'
        return f'background-color: {color}'
    def CT_highlighter(val):
        color = '#D3F6EC' if val > 0 else 'transparent'
        return f'background-color: {color}'
    def token_highlighter(val):
        color = '#D3F6EC' if weight_token(val) > 0 else 'transparent'
        return f'background-color: {color}'
        
    # Sum the boolean conditions after converting to int
    # Base boolean columns
    cols = [
        'tegangan_kecil', 'cosphi_kecil', 'keypad_bermasalah',
        'indikator_temper', 'segel_bermasalah', 'opsi_relay', 'LCD_bermasalah'
    ]

    # Include CT_weight if JML_CT exists
    if 'JML_CT' in df_pra.columns:
        cols.append('CT_weight')
    # Always sum available columns; convert to numeric once
    df_pra['Jumlah Potensi TO'] = df_pra[cols].apply(pd.to_numeric, errors='coerce').sum(axis=1)
    # Add token if available
    if 'Keterangan_Token' in df_pra.columns:
        df_pra['Jumlah Potensi TO'] += pd.to_numeric(df_pra['token_weight'], errors='coerce')

    df_pra = df_pra[df_pra['Jumlah Potensi TO']>=n_indikator]
    n_pto = len(df_pra)

    ## METRIC
    kol1, kol2 = st.columns((1, 1))    
    df_pra['Google_Maps_Link'] = 'https://maps.google.com/?q=' + df_pra['KOORDINAT Y'].astype(str) + ',' + df_pra['KOORDINAT X'].astype(str)
    kol1.metric(label=f"Total Pelanggan di Analisis Periode {this_month}", value=f"{n_data:,}")
    kol2.metric(label="Target Operasi Memenuhi Kriteria", value=f"{n_pto:,}")

    # Style Highlighter
    IMPORTANT_COL = ['IDPEL', 'NAMA', 'ALAMAT', 'TARIF', 'DAYA', 'PETUGAS', 'KDBACA', 'TEGANGAN', 'COSPHI', 'SEGEL', 'LCD', 'KEYPAD', 'INDI_TEMPER', 'RELAY', 'Jumlah Potensi TO', 'Google_Maps_Link']
    IMPORTANT_COL2 = ['IDPEL', 'TARIF', 'DAYA', 'PETUGAS', 'KDBACA', 'TEGANGAN', 'COSPHI', 'SEGEL', 'LCD', 'KEYPAD', 'INDI_TEMPER', 'RELAY']
    if 'JML_CT' in df_pra.columns:
        IMPORTANT_COL.extend(['JML_CT', 'Keterangan_CT'])
        IMPORTANT_COL2.extend(['JML_CT', 'Keterangan_CT'])
    if 'Keterangan_Token' in df_pra.columns:
        IMPORTANT_COL.append('Keterangan_Token')
        IMPORTANT_COL2.append('Keterangan_Token')
    IMPORTANT_COL2.append('Jumlah Potensi TO')
        
    df_pra_style = df_pra.sort_values(by='Jumlah Potensi TO', ascending=False).reset_index(drop=True)[IMPORTANT_COL2].head(n_show)
    df_pra_style = df_pra_style.style.applymap(tegangan_rendah, subset=['TEGANGAN'])
    df_pra_style = df_pra_style.applymap(cosphi_rendah, subset=['COSPHI'])
    df_pra_style = df_pra_style.applymap(segel_highlighter, subset=['SEGEL'])

    df_pra_style = df_pra_style.applymap(lcd_highlighter, subset=['LCD'])
    df_pra_style = df_pra_style.applymap(keypad_highlighter, subset=['KEYPAD'])
    df_pra_style = df_pra_style.applymap(indi_temper_highlighter, subset=['INDI_TEMPER'])
    df_pra_style = df_pra_style.applymap(relay_highlighter, subset=['RELAY'])
    if 'JML_CT' in df_pra.columns:
        df_pra_style = df_pra_style.applymap(CT_highlighter, subset=['JML_CT'])
    if 'Keterangan_Token' in df_pra.columns:
        df_pra_style = df_pra_style.applymap(token_highlighter, subset=['Keterangan_Token'])
    df_pra_style = df_pra_style.format({
            'IDPEL': '{:.0f}',
            'DAYA': '{:.0f}', 
            'TEGANGAN': '{:.0f}'
        })
    st.subheader(f"Top {n_show} Rekomendasi Target Operasi")
    st.dataframe(df_pra_style, use_container_width=True, hide_index=True)

    # Buat Download CSV
    st.download_button(
        "Tekan untuk Download Data",
        df_pra[IMPORTANT_COL].to_csv(index=False, sep=';').encode('utf-8'),
        "hasil_analisis_prabayar.csv",
        "text/csv",
        key='download-csv'
    )
    
# Prabayar Properties - Operating
def operate_pra():
    st.markdown("""
    <h3>Otomatisasi Target Operasi <strong>Pelanggan Prabayar</strong></h4>
    """, unsafe_allow_html=True)
    with st.expander(label="Input Data (Semua Data Wajib di Konversi ke xlsx)"):
        c1_pra, c2_pra, c3_pra = st.columns(3)
        # File Loader Step 1
        uploaded_input = c1_pra.file_uploader(
            "Inputkan Data Prabayar x+1 (Wajib)", type=['xlsx']
        )
        placeholder1 = c1_pra.empty()
        placeholder1 = c1_pra.warning('⚠️ Data Belum di Upluod')
        # File Loader Step 2
        opsional_temper  = c2_pra.file_uploader(
            "Data Permohonan Clear Temper x (Opsional)", type=['xlsx']
        )
        placeholder2 = c2_pra.empty()
        # File Loader Step 3
        opsional_token  = c3_pra.file_uploader(
            "Data Tidak Membeli Token x (Opsional)", type=["xlsx"]
        )
        placeholder3 = c3_pra.empty()

    if uploaded_input:
        cols = ["NO","UNITUP","IDPEL","BLTH","NAMA","ALAMAT","NO METER","TARIF","KDPT","DAYA","KDRBM","LANGKAH","SISIPAN","KDKELOMPOK","TGLBACA","KDBACA","PETUGAS","TEGANGAN","ARUS","COSPHI","TARIF INDEX","POWER LIMIT","KWH KUMULATIF","INDIKATOR","KWH SISA","STATUS TEMPER","TUTUP METER","SEGEL","LCD","KEYPAD","JML_TERMINAL","INDI_TEMPER","RELAY","KOORDINAT X","KOORDINAT Y","AKURASI"]
        try: 
            df_pra = read_excel(uploaded_input, skiprows=1, usecols=cols)
        except Exception as e: 
            c1_pra.warning("⚠️ Gagal Membaca Data")
        # take one of UNITUP and BLTH
        unitup = str(int(df_pra.loc[0, "UNITUP"]))[:3]
        blth = str(int(df_pra.loc[0, "BLTH"]))
        df_pra["KDBACA"] = df_pra['KDBACA'].apply(convert_kode_baca)
        df_pra = df_pra[df_pra[["UNITUP", "IDPEL", "BLTH", "NO METER"]].notna().all(axis=1)]
        placeholder1.success("✅ Data telah selesai di Proses")
        if opsional_temper: 
            try:
                df_temper = read_excel(opsional_temper)
                unitup_temper = str(int(df_temper.loc[0, "UNITUP"]))[:3]
                blth_temper = str(int(df_temper.loc[0, "THBL"]))
                if (unitup_temper==unitup) & (blth_temper==fix_blth_1(blth)):
                    placeholder2.success("✅ Data sesuai, dan telah selesai di Proses")
                    df_temper = df_temper[['IDPEL', 'JML_CT', 'KETERANGAN']].rename({
                        'KETERANGAN': 'Keterangan_CT'
                    }, axis=1)
                    df_pra=df_pra.merge(right=df_temper, on='IDPEL', how='left')
                    df_pra['JML_CT'] = df_pra['JML_CT'].fillna(0).astype(int)
                else:
                    placeholder2.warning("Data tidak sesuai, Data 1 merupakan periode + 1")
                    temp = pd.DataFrame({
                        'Label': ['UNIT', 'BLTH'],
                        'Prabayar': [unitup, blth], 
                        'CT': [unitup_temper, blth_temper]
                    })
                    temp = temp.set_index('Label')
                    c2_pra.write(temp)
            except Exception as e:
                c2_pra.warning("⚠️ Gagal Membaca Data")
        if opsional_token: 
            try:
                df_token = read_excel(opsional_token)
                unitup_token = str(int(df_token.loc[0, "UNITUP"]))[:3]
                blth_token = str(int(df_token.loc[0, "THBL"]))
                if (unitup_token==unitup) & (blth_token==fix_blth_1(blth)):
                    placeholder3.success("✅ Data sesuai, dan telah selesai di Proses")
                    df_token = df_token[['IDPEL', 'KETERANGAN']].rename({
                        'KETERANGAN': 'Keterangan_Token'
                    }, axis=1)
                    df_pra=df_pra.merge(right=df_token, on='IDPEL', how='left')
                    df_pra.replace({np.nan: None})
                else:
                    placeholder2.warning("Data tidak sesuai, Data 1 merupakan periode + 1")
                    temp = pd.DataFrame({
                        'Label': ['UNIT', 'BLTH'],
                        'Prabayar': [unitup, blth], 
                        'Token': [unitup_token, blth_token]
                    })
                    temp = temp.set_index('Label')
                    c3_pra.write(temp)
            except Exception as e:
                c3_pra.warning("⚠️ Gagal Membaca Data")
        # Once Done, processing it 
        dashboarding_pra(df_pra)
    st.divider()

##### MAIN
page = st.query_params.get("page", "amr")
placenavbar = st.empty()
placenavbar.html(f"""
<div class="navbar">
    <div class="navbar-logo">
        <img src="https://github.com/user-attachments/assets/15349020-096c-4c47-8022-749a6b0593c1">
        <div class="navbar-title">Sistem Otomatisasi Penyusunan Target Operasi P2TL V2</div>
    </div>
    <div class="navbar-menu">
        <a href="/?page=amr" class="{'active' if page=='amr' else''}" target="_self">AMR</a>
        <a href="/?page=loadprofile" class="{'active' if page=='loadprofile' else''}" target="_self">Load Profile</a>
        <a href="/?page=pascabayar" class="{'active' if page=='pascabayar' else''}" target="_self">Pascabayar</a>
        <a href="/?page=prabayar" class="{'active' if page=='prabayar' else''}" target="_self">Prabayar</a>
        <a href="https://autogen-pln.site/" target="_self">Beranda</a>
    </div>
</div>
""") 
# Kata-kata Welcoming
def welcoming(user):
    cas1, cas2 = st.columns((7, 1))
    cas1.markdown(f"""
        <p style='font-size:40px; font-weight:bold; margin-bottom:0'>
            Selamat Datang <span style='font-size:30px; font-weight:normal'> {user} 👋🏻</span>
        </p>
    """, unsafe_allow_html=True)
    with cas2:
        st.write("")
        if st.button("Logout"):
            st.logout()
    st.divider()

if 'logged_in' not in st.session_state:
    st.session_state['logged_in'] = False
placeLog = st.empty()
if st.session_state['logged_in'] == False:
    with placeLog.container():
        password = st.text_input("Inputkan Kata Kunci / Password untuk Akses", key="password", type="password")
        if ((password == user_pass) | (password == user_pass2)):
            if (password == user_pass):
                user = "Fauzi Hidayat"
            else:
                user = "PLN'ers"
            st.session_state["logged_in"] = True
            placeLog.success("Berhasil Login !")
            sleep(0.5)
            placeLog.empty()
        elif (len(password) < 1):
            st.info("Kunjungi dokumentasi kami, atau hubungi pihak terkait...")
        else:
            st.warning("Password Salah !")

if __name__ == "__main__":
    if st.session_state['logged_in'] == True:
        # welcoming(user)
        if page == "amr":
            operate_amr()
        elif page == "pascabayar":
            operate_pasca()
        elif page == "prabayar":
            operate_pra()
        elif page == "loadprofile":
            operate_loadProfile()

    # Setelah semua layout selesai
    with st.container():
        st.markdown("""
        <footer class="footer">
            <div class="footer-container">
                <div class="footer-column">
                    <img src="https://github.com/user-attachments/assets/15349020-096c-4c47-8022-749a6b0593c1" width="300" height="300" alt="Ini Logo P2TL">
                </div>
                <div class="footer-column">
                    <h3>Auto-Generate TO P2TL</h3>
                    <h5>Produk Inovasi UP3 Menteng</h5>
                    <p>l. M.I. Ridwan Rais No.1, RT.7/RW.2, Gambir, Kecamatan Gambir, Kota Jakarta Pusat, Daerah Khusus Ibukota Jakarta 10110</p>
                    <p><strong><li>Muhammad Fauzi Hidayat</li></strong></p>
                    <p><strong><li>Muhammad Refi Hidayat</li></strong></p>
                    <p><strong><li>Rahardian Anggada Setya Putra</li></strong></p>
                </div>
                <div class="footer-column">
                    <h3>Otomatisasi Target Operasi</h3>
                    <ul>
                        <li>› Pelanggan Automatic Meter Reading</li>
                        <li>› Pelanggan Pascabayar</li>
                        <li>› Pelanggan Prabayar</li>
                    </ul>
                </div>
            </div>
            <div class="footer-bottom">
                <p>©Copyright 2026 UP3 Menteng Version 2</p>
            </div>
        </footer>
        """, unsafe_allow_html=True)